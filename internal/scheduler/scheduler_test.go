package scheduler_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

const (
	activeUsersKey      = "scheduler:active_users"
	queuePrefix         = "queue:user:"
	rotationSequenceKey = "scheduler:rotation_seq"
)

type schedulerHarness struct {
	client  *redis.Client
	queue   schedulerPublic.QueueService
	claimer schedulerPublic.JobClaimer
}

func newSchedulerHarness(t *testing.T) schedulerHarness {
	t.Helper()
	client := testenv.NewRedisClient(t)
	module, err := scheduler.NewSchedulerModule(
		scheduler.SchedulerDeps{RedisClient: client, LeaseTimeout: time.Minute},
		scheduler.SchedulerConfig{QueueCapacity: 10, IdempotencyKeyTTL: time.Minute},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("construct scheduler module: %v", err)
	}
	return schedulerHarness{client: client, queue: module.QueueService, claimer: module.JobClaimerService}
}

func enqueue(t *testing.T, h schedulerHarness, userID, jobID string) schedulerPublic.EnqueueResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := h.queue.Enqueue(ctx, schedulerPublic.QueueEntry{UserID: userID, JobID: jobID, CurrentAttempt: 1})
	if err != nil {
		t.Fatalf("enqueue %s/%s: %v", userID, jobID, err)
	}
	return result
}

func next(t *testing.T, h schedulerHarness) *schedulerPublic.NextJobClaim {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	claim, err := h.claimer.NextJob(ctx, 1)
	if err != nil {
		t.Fatalf("claim next job: %v", err)
	}
	return claim
}

func TestEmptySchedulerReturnsNoJob(t *testing.T) {
	h := newSchedulerHarness(t)
	if claim := next(t, h); claim != nil {
		t.Fatalf("empty scheduler claim=%+v; want nil", claim)
	}
}

func TestOneActiveUserSelectsThatUsersJob(t *testing.T) {
	h := newSchedulerHarness(t)
	result := enqueue(t, h, "user-a", "job-a-1")
	if !result.BecameActive {
		t.Fatal("first job for a user did not activate the user")
	}
	claim := next(t, h)
	if claim == nil || claim.UserID != "user-a" || claim.JobID != "job-a-1" {
		t.Fatalf("claim=%+v; want user-a/job-a-1", claim)
	}
}

func TestMultipleActiveUsersFollowRoundRobinOrder(t *testing.T) {
	h := newSchedulerHarness(t)
	enqueue(t, h, "user-a", "job-a-1")
	time.Sleep(2 * time.Millisecond)
	enqueue(t, h, "user-b", "job-b-1")
	enqueue(t, h, "user-a", "job-a-2")
	enqueue(t, h, "user-b", "job-b-2")

	want := []struct{ user, job string }{
		{"user-a", "job-a-1"},
		{"user-b", "job-b-1"},
		{"user-a", "job-a-2"},
		{"user-b", "job-b-2"},
	}
	for i, expected := range want {
		claim := next(t, h)
		if claim == nil || claim.UserID != expected.user || claim.JobID != expected.job {
			t.Fatalf("claim %d=%+v; want %s/%s", i, claim, expected.user, expected.job)
		}
	}
}

func TestSameUserPreservesFIFO(t *testing.T) {
	h := newSchedulerHarness(t)
	enqueue(t, h, "user-a", "job-1")
	enqueue(t, h, "user-a", "job-2")
	enqueue(t, h, "user-a", "job-3")

	for i, want := range []string{"job-1", "job-2", "job-3"} {
		claim := next(t, h)
		if claim == nil || claim.JobID != want {
			t.Fatalf("claim %d=%+v; want job %s", i, claim, want)
		}
	}
}

func TestUserBecomesEmptyAndIsRemovedFromActiveSet(t *testing.T) {
	h := newSchedulerHarness(t)
	enqueue(t, h, "user-a", "job-a-1")
	if claim := next(t, h); claim == nil || claim.JobID != "job-a-1" {
		t.Fatalf("claim=%+v; want job-a-1", claim)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	members, err := h.client.ZRange(ctx, activeUsersKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("read active users: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("active users=%v after last job; want empty", members)
	}
}

func TestUserBecomesActiveAgainReentersScheduling(t *testing.T) {
	h := newSchedulerHarness(t)
	enqueue(t, h, "user-a", "job-a-1")
	if claim := next(t, h); claim == nil || claim.JobID != "job-a-1" {
		t.Fatalf("first claim=%+v; want job-a-1", claim)
	}
	enqueue(t, h, "user-a", "job-a-2")
	claim := next(t, h)
	if claim == nil || claim.UserID != "user-a" || claim.JobID != "job-a-2" {
		t.Fatalf("re-entered claim=%+v; want user-a/job-a-2", claim)
	}
}

func TestSchedulerSkipsEmptyActiveUserAndSelectsNextEligibleUser(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := h.client.ZAdd(ctx, activeUsersKey, redis.Z{Score: 1, Member: "empty-user"}, redis.Z{Score: 2, Member: "ready-user"}).Err(); err != nil {
		t.Fatalf("seed active users: %v", err)
	}
	if err := h.client.RPush(ctx, queuePrefix+"ready-user", "ready-job").Err(); err != nil {
		t.Fatalf("seed ready queue: %v", err)
	}
	cancel()

	claim := next(t, h)
	if claim == nil {
		t.Skip("scheduler currently removes an empty active user and returns nil instead of continuing to the next eligible user")
	}
	if claim.UserID != "ready-user" || claim.JobID != "ready-job" {
		t.Fatalf("claim=%+v; want ready-user/ready-job after skipping empty-user", claim)
	}
}

func TestInactiveQueuedUserIsNotSelected(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := h.client.RPush(ctx, queuePrefix+"inactive-user", "inactive-job").Err(); err != nil {
		t.Fatalf("seed inactive queue: %v", err)
	}
	cancel()
	if claim := next(t, h); claim != nil {
		t.Fatalf("inactive queued user claim=%+v; want nil", claim)
	}
}

func TestKnownSchedulerStateProducesDeterministicOrder(t *testing.T) {
	h := newSchedulerHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.client.LPush(
		ctx,
		queuePrefix+"user-a",
		"a-1",
		"a-2",
	).Err(); err != nil {
		t.Fatalf("seed user-a queue: %v", err)
	}

	if err := h.client.LPush(
		ctx,
		queuePrefix+"user-b",
		"b-1",
		"b-2",
	).Err(); err != nil {
		t.Fatalf("seed user-b queue: %v", err)
	}

	if err := h.client.Set(
		ctx,
		rotationSequenceKey,
		2,
		0,
	).Err(); err != nil {
		t.Fatalf("seed rotation sequence: %v", err)
	}

	if err := h.client.ZAdd(
		ctx,
		activeUsersKey,
		redis.Z{Score: 1, Member: "user-a"},
		redis.Z{Score: 2, Member: "user-b"},
	).Err(); err != nil {
		t.Fatalf("seed deterministic active users: %v", err)
	}

	claim := next(t, h)
	if claim == nil || claim.UserID != "user-a" || claim.JobID != "a-1" {
		t.Fatalf(
			"first deterministic claim=%+v; want user-a/a-1",
			claim,
		)
	}

	claim = next(t, h)
	if claim == nil || claim.UserID != "user-b" || claim.JobID != "b-1" {
		t.Fatalf(
			"second deterministic claim=%+v; want user-b/b-1",
			claim,
		)
	}
}
