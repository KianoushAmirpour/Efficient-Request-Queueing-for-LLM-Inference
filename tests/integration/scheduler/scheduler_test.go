package scheduler_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

const (
	queuePrefix    = "queue:user:"
	activeUsersKey = "scheduler:active_users"
	processingKey  = "processing_jobs"
)

type harness struct {
	redis   *redis.Client
	queue   schedulerPublic.QueueService
	claimer schedulerPublic.JobClaimer
}

func newHarness(t *testing.T, capacity int) harness {
	t.Helper()
	client := testenv.NewRedisClient(t)
	module, err := scheduler.NewSchedulerModule(
		scheduler.SchedulerDeps{RedisClient: client, LeaseTimeout: time.Minute},
		scheduler.SchedulerConfig{QueueCapacity: capacity, IdempotencyKeyTTL: time.Minute},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("construct scheduler module: %v", err)
	}
	return harness{redis: client, queue: module.QueueService, claimer: module.JobClaimerService}
}

func enqueue(t *testing.T, h harness, userID, jobID string) schedulerPublic.EnqueueResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := h.queue.Enqueue(ctx, schedulerPublic.QueueEntry{UserID: userID, JobID: jobID, CurrentAttempt: 1})
	if err != nil {
		t.Fatalf("enqueue %s/%s: %v", userID, jobID, err)
	}
	return result
}

func claim(t *testing.T, h harness, workerID int) *schedulerPublic.NextJobClaim {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := h.claimer.NextJob(ctx, workerID)
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	return result
}

func redisInt(t *testing.T, h harness, command func(context.Context) (int64, error)) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := command(ctx)
	if err != nil {
		t.Fatalf("read Redis invariant: %v", err)
	}
	return value
}

func TestSchedulerIntegrationEnqueueCreatesPerUserQueue(t *testing.T) {
	h := newHarness(t, 20)
	enqueue(t, h, "user-a", "a-1")
	enqueue(t, h, "user-a", "a-2")
	enqueue(t, h, "user-b", "b-1")

	if got := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.LLen(ctx, queuePrefix+"user-a").Result() }); got != 2 {
		t.Fatalf("user-a queue length=%d; want 2", got)
	}
	if got := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.LLen(ctx, queuePrefix+"user-b").Result() }); got != 1 {
		t.Fatalf("user-b queue length=%d; want 1", got)
	}
}

func TestSchedulerIntegrationActivatesUserOnce(t *testing.T) {
	h := newHarness(t, 20)
	if result := enqueue(t, h, "user-a", "a-1"); !result.BecameActive {
		t.Fatal("first enqueue did not activate user")
	}
	if result := enqueue(t, h, "user-a", "a-2"); result.BecameActive {
		t.Fatal("second enqueue reactivated an already active user")
	}
	if got := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.ZCard(ctx, activeUsersKey).Result() }); got != 1 {
		t.Fatalf("active user count=%d; want 1", got)
	}
}

func TestSchedulerIntegrationDequeueRemovesCorrectJobAndTracksProcessing(t *testing.T) {
	h := newHarness(t, 20)
	enqueue(t, h, "user-a", "a-1")
	enqueue(t, h, "user-a", "a-2")
	got := claim(t, h, 1)
	if got == nil || got.UserID != "user-a" || got.JobID != "a-1" {
		t.Fatalf("claim=%+v; want user-a/a-1", got)
	}
	if length := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.LLen(ctx, queuePrefix+"user-a").Result() }); length != 1 {
		t.Fatalf("remaining user queue length=%d; want 1", length)
	}
	if score := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.ZCard(ctx, processingKey).Result() }); score != 1 {
		t.Fatalf("processing job count=%d; want 1", score)
	}
}

func TestSchedulerIntegrationFairnessAndFIFOForUnevenQueues(t *testing.T) {
	h := newHarness(t, 20)
	// Build the scenario through the public enqueue path. This initializes the
	// active-user set and rotation sequence consistently with production.
	for _, entry := range []struct {
		userID string
		jobIDs []string
	}{
		{userID: "user-a", jobIDs: []string{"A1", "A2", "A3", "A4"}},
		{userID: "user-b", jobIDs: []string{"B1", "B2"}},
		{userID: "user-c", jobIDs: []string{"C1", "C2"}},
	} {
		for _, jobID := range entry.jobIDs {
			enqueue(t, h, entry.userID, jobID)
		}
	}

	want := []string{"A1", "B1", "C1", "A2", "B2", "C2", "A3", "A4"}
	for i, expected := range want {
		got := claim(t, h, i+1)
		if got == nil || got.JobID != expected {
			t.Fatalf("claim %d=%+v; want %s", i, got, expected)
		}
	}
}

func TestSchedulerIntegrationRoundRobinEqualQueuesOfOneHundred(t *testing.T) {
	h := newHarness(t, 100)
	for _, userID := range []string{"user-a", "user-b", "user-c"} {
		for jobNumber := 1; jobNumber <= 100; jobNumber++ {
			enqueue(t, h, userID, jobIDFor(userID, jobNumber))
		}
	}

	for round := 1; round <= 100; round++ {
		for _, userID := range []string{"user-a", "user-b", "user-c"} {
			want := jobIDFor(userID, round)
			got := claim(t, h, round)
			if got == nil || got.UserID != userID || got.JobID != want {
				t.Fatalf("round %d claim for %s=%+v; want %s/%s", round, userID, got, userID, want)
			}
		}
	}
}

func TestSchedulerIntegrationRoundRobinStopsForExhaustedUsers(t *testing.T) {
	h := newHarness(t, 100)
	for jobNumber := 1; jobNumber <= 100; jobNumber++ {
		enqueue(t, h, "user-a", jobIDFor("user-a", jobNumber))
	}
	for _, userID := range []string{"user-b", "user-c"} {
		for jobNumber := 1; jobNumber <= 2; jobNumber++ {
			enqueue(t, h, userID, jobIDFor(userID, jobNumber))
		}
	}

	for round := 1; round <= 2; round++ {
		for _, userID := range []string{"user-a", "user-b", "user-c"} {
			want := jobIDFor(userID, round)
			got := claim(t, h, round)
			if got == nil || got.UserID != userID || got.JobID != want {
				t.Fatalf("round %d claim for %s=%+v; want %s/%s", round, userID, got, userID, want)
			}
		}
	}

	for jobNumber := 3; jobNumber <= 100; jobNumber++ {
		got := claim(t, h, jobNumber)
		want := jobIDFor("user-a", jobNumber)
		if got == nil || got.UserID != "user-a" || got.JobID != want {
			t.Fatalf("post-exhaustion claim %d=%+v; want user-a/%s", jobNumber, got, want)
		}
	}
}

func jobIDFor(userID string, jobNumber int) string {
	return fmt.Sprintf("%s-job-%03d", userID, jobNumber)
}

func TestSchedulerIntegrationEmptyUserIsRemoved(t *testing.T) {
	h := newHarness(t, 20)
	enqueue(t, h, "user-a", "a-1")
	if got := claim(t, h, 1); got == nil || got.JobID != "a-1" {
		t.Fatalf("claim=%+v; want a-1", got)
	}
	if got := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.ZCard(ctx, activeUsersKey).Result() }); got != 0 {
		t.Fatalf("active users after final dequeue=%d; want 0", got)
	}
}

func TestSchedulerIntegrationConcurrentEnqueueAndDequeuePreservesInvariants(t *testing.T) {
	h := newHarness(t, 200)
	const total = 100
	start := make(chan struct{})
	var enqueued atomic.Int32
	var claimed atomic.Int32
	seen := make(map[string]struct{}, total)
	var seenMu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-start
			for claimed.Load() < total {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				job, err := h.claimer.NextJob(ctx, workerID)
				cancel()
				if err != nil {
					t.Errorf("concurrent claim: %v", err)
					return
				}
				if job == nil {
					time.Sleep(time.Millisecond)
					continue
				}
				seenMu.Lock()
				if _, exists := seen[job.JobID]; exists {
					seenMu.Unlock()
					t.Errorf("job %q claimed more than once", job.JobID)
					return
				}
				seen[job.JobID] = struct{}{}
				seenMu.Unlock()
				claimed.Add(1)
			}
		}(i)
	}

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := h.queue.Enqueue(ctx, schedulerPublic.QueueEntry{UserID: "user-" + string(rune('a'+i%4)), JobID: "job-" + string(rune('a'+i)), CurrentAttempt: 1})
			cancel()
			if err != nil {
				t.Errorf("concurrent enqueue: %v", err)
				return
			}
			enqueued.Add(1)
		}(i)
	}
	close(start)
	wg.Wait()

	if enqueued.Load() != total || claimed.Load() != total {
		t.Fatalf("enqueued=%d claimed=%d; want %d each", enqueued.Load(), claimed.Load(), total)
	}
	if got := redisInt(t, h, func(ctx context.Context) (int64, error) { return h.redis.ZCard(ctx, processingKey).Result() }); got != total {
		t.Fatalf("processing jobs=%d; want %d", got, total)
	}
	if got := len(seen); got != total {
		t.Fatalf("unique claimed jobs=%d; want %d", got, total)
	}
}
