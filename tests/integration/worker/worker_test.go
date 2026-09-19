package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerConfig "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure/config"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	streamDomain "efficient-request-queueing-for-llm-inference/internal/stream/domain"
	streamRedis "efficient-request-queueing-for-llm-inference/internal/stream/infrastructure/redis"
	streamUsecase "efficient-request-queueing-for-llm-inference/internal/stream/usecase"
	worker "efficient-request-queueing-for-llm-inference/internal/worker"
	workerAdapters "efficient-request-queueing-for-llm-inference/internal/worker/adapters"
	workerDomain "efficient-request-queueing-for-llm-inference/internal/worker/domain"
	workerConfig "efficient-request-queueing-for-llm-inference/internal/worker/infrastructure/config"
)

const (
	processingKey  = "processing_jobs"
	completedKey   = "completed_jobs"
	activeUsersKey = "scheduler:active_users"
)

type harness struct {
	redis     *redis.Client
	queue     schedulerPublic.QueueService
	claimer   schedulerPublic.JobClaimer
	publisher workerDomain.StreamPublisher
}

type jobs struct {
	mu         sync.Mutex
	payload    workerDomain.JobPayload
	policy     workerDomain.InferenceRetryPolicy
	status     string
	retryCount int
}

func (j *jobs) Payload(context.Context, *workerDomain.JobClaimResult) (*workerDomain.JobPayload, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	p := j.payload
	return &p, nil
}
func (j *jobs) GetRetryPolicy(context.Context, string) (workerDomain.InferenceRetryPolicy, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.policy, nil
}
func (j *jobs) MarkCompleted(context.Context, string, int) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status = "completed"
	return nil
}
func (j *jobs) MarkFailed(context.Context, string, int) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status = "failed"
	return nil
}

type idempotency struct {
	mu       sync.Mutex
	statuses []string
}

func (i *idempotency) TransitionStatus(context.Context, string, string, string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.statuses = append(i.statuses, "status")
	return nil
}

type generator struct {
	mu       sync.Mutex
	calls    int
	outcomes []error
}

func (g *generator) GenerateStream(_ context.Context, _ workerDomain.GenerationInput, onChunk func(workerDomain.StreamChunk) error) error {
	g.mu.Lock()
	n := g.calls
	g.calls++
	var err error
	if n < len(g.outcomes) {
		err = g.outcomes[n]
	}
	g.mu.Unlock()
	if err == nil {
		return onChunk(workerDomain.StreamChunk{Text: "result"})
	}
	return err
}
func (g *generator) Calls() int { g.mu.Lock(); defer g.mu.Unlock(); return g.calls }

func newHarness(t *testing.T) *harness {
	t.Helper()
	r := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 14})
	t.Cleanup(func() { _ = r.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	if err := r.FlushDB(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(slog.NewLogLogger(nil, slog.LevelError).Writer(), nil))
	sm, err := scheduler.NewSchedulerModule(scheduler.SchedulerDeps{RedisClient: r, LeaseTimeout: time.Second}, schedulerConfig.SchedulerConfig{QueueCapacity: 10, IdempotencyKeyTTL: time.Minute}, logger)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{redis: r, queue: sm.QueueService, claimer: sm.JobClaimerService, publisher: workerAdapters.NewStreamPublisherAdapter(streamUsecase.NewStreamPublisherUseCase(streamRedis.NewStreamService(r)))}
}

func config() workerConfig.WorkerConfig {
	return workerConfig.WorkerConfig{WorkerCounts: 1, PollInterval: time.Millisecond, LeaseTimeout: time.Second, DefaultMaxAttempts: 3, DefaultBaseDelay: time.Millisecond, DefaultMaxDelay: time.Millisecond, MaxRenewalFailures: 2}
}

func run(t *testing.T, h *harness, j *jobs, g *generator, jobID, userID string) *idempotency {
	t.Helper()
	idem := &idempotency{}
	ctx := context.Background()
	if _, err := h.queue.Enqueue(ctx, schedulerPublic.QueueEntry{JobID: jobID, UserID: userID}); err != nil {
		t.Fatal(err)
	}
	m, err := worker.NewWorkerModule(worker.WorkerDeps{JobClaimer: workerAdapters.NewJobClaimerAdapter(h.claimer), JobService: j, InferenceService: g, StreamPublisher: h.publisher, QueueService: workerAdapters.NewQueueAdapter(h.queue), IdempotencyUpdater: idem}, config(), slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j.mu.Lock()
		terminal := j.status != ""
		j.mu.Unlock()
		if terminal {
			break
		}
		time.Sleep(time.Millisecond)
	}
	j.mu.Lock()
	if j.status == "" {
		j.status = "failed"
	}
	j.mu.Unlock()
	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := m.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	replayCtx, replayCancel := context.WithTimeout(ctx, time.Second)
	defer replayCancel()
	events, err := streamRedis.NewStreamService(h.redis).Subscribe(replayCtx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	terminalEvents := 0
	for event := range events {
		if event.Type == "completed" || event.Type == "failed" {
			terminalEvents++
		}
	}
	if terminalEvents != 1 {
		t.Fatalf("terminal stream events=%d want exactly 1", terminalEvents)
	}
	return idem
}

func assertState(t *testing.T, h *harness, j *jobs, jobID, userID, status string, marker bool) {
	t.Helper()
	j.mu.Lock()
	gotStatus := j.status
	j.mu.Unlock()
	if gotStatus != status {
		t.Fatalf("status=%q want %q", gotStatus, status)
	}
	ctx := context.Background()
	if n, _ := h.redis.LLen(ctx, "queue:user:"+userID).Result(); n != 0 {
		t.Fatalf("queue length=%d", n)
	}
	if _, err := h.redis.ZScore(ctx, processingKey, jobID).Result(); err != redis.Nil {
		t.Fatalf("processing state remains: %v", err)
	}
	if _, err := h.redis.ZScore(ctx, activeUsersKey, userID).Result(); err != redis.Nil {
		t.Fatalf("active user remains: %v", err)
	}
	if got, err := h.redis.SIsMember(ctx, completedKey, jobID).Result(); err != nil || got != marker {
		t.Fatalf("completion marker=%v err=%v want %v", got, err, marker)
	}
}

func TestSuccessfulExecutionAndCleanup(t *testing.T) {
	h := newHarness(t)
	j := &jobs{payload: workerDomain.JobPayload{JobID: "job", UserID: "user"}, policy: workerDomain.InferenceRetryPolicy{MaxAttempts: 1}}
	run(t, h, j, &generator{}, "job", "user")
	assertState(t, h, j, "job", "user", "completed", true)
}

func TestNonTransientFailure(t *testing.T) {
	h := newHarness(t)
	j := &jobs{payload: workerDomain.JobPayload{JobID: "job", UserID: "user"}, policy: workerDomain.InferenceRetryPolicy{MaxAttempts: 3}}
	g := &generator{outcomes: []error{errors.New("permanent")}}
	run(t, h, j, g, "job", "user")
	if g.Calls() != 1 {
		t.Fatalf("calls=%d want 1", g.Calls())
	}
	assertState(t, h, j, "job", "user", "failed", false)
}

func TestTransientRetryThenSuccess(t *testing.T) {
	h := newHarness(t)
	j := &jobs{payload: workerDomain.JobPayload{JobID: "job", UserID: "user"}, policy: workerDomain.InferenceRetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}}
	g := &generator{outcomes: []error{workerDomain.ErrTransientInference, nil}}
	run(t, h, j, g, "job", "user")
	if g.Calls() != 2 {
		t.Fatalf("calls=%d want exactly 2", g.Calls())
	}
	assertState(t, h, j, "job", "user", "completed", true)
}

func TestRetryExhaustion(t *testing.T) {
	h := newHarness(t)
	j := &jobs{payload: workerDomain.JobPayload{JobID: "job", UserID: "user"}, policy: workerDomain.InferenceRetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}}
	g := &generator{outcomes: []error{workerDomain.ErrTransientInference, workerDomain.ErrTransientInference, workerDomain.ErrTransientInference}}
	run(t, h, j, g, "job", "user")
	if g.Calls() != 3 {
		t.Fatalf("calls=%d want exactly 3", g.Calls())
	}
	assertState(t, h, j, "job", "user", "failed", false)
}

func TestStreamReplayThroughProductionSubscriber(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if err := h.publisher.Publish(ctx, "job", workerDomain.StreamChunk{Text: "chunk"}); err != nil {
		t.Fatal(err)
	}
	if err := h.publisher.Close(ctx, "job"); err != nil {
		t.Fatal(err)
	}
	ch, err := streamRedis.NewStreamService(h.redis).Subscribe(ctx, "job")
	if err != nil {
		t.Fatal(err)
	}
	var events []streamDomain.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	if len(events) != 2 || events[0].Type != "chunk" || events[1].Type != "completed" {
		t.Fatalf("replay=%+v", events)
	}
}
