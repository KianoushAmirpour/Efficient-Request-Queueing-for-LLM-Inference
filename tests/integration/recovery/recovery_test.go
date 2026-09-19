package recovery_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	inferenceConfig "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
	idempotency "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/idempotency"
	job "efficient-request-queueing-for-llm-inference/internal/job"
	jobDomain "efficient-request-queueing-for-llm-inference/internal/job/domain"
	jobConfig "efficient-request-queueing-for-llm-inference/internal/job/infrastructure/config"
	jobPublic "efficient-request-queueing-for-llm-inference/internal/job/public"
	recoveryAdapters "efficient-request-queueing-for-llm-inference/internal/recovery/adapters"
	recovery "efficient-request-queueing-for-llm-inference/internal/recovery/usecase"
	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerConfig "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure/config"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	stream "efficient-request-queueing-for-llm-inference/internal/stream"
	streamPorts "efficient-request-queueing-for-llm-inference/internal/stream/ports"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

const (
	redisDB       = 12
	processingKey = "processing_jobs"
	completedKey  = "completed_jobs"
	queuePrefix   = "queue:user:"
)

type users struct{}

func (users) GetUserPolicy(context.Context, string) (jobDomain.UserPolicy, error) {
	return jobDomain.UserPolicy{Tier: "free"}, nil
}

type tokens struct{}

func (tokens) ValidateAccessToken(string) (string, string, error) { return "recovery", "user", nil }

type harness struct {
	r        *redis.Client
	pg       *pgxpool.Pool
	jobs     jobPublic.JobService
	reader   jobPublic.JobReader
	queue    schedulerPublic.QueueService
	claimer  schedulerPublic.JobClaimer
	service  *recovery.RecoveryService
	serviceB *recovery.RecoveryService
	idem     *idempotency.RedisStore
}

func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 4*time.Second)
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	r := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: redisDB})
	t.Cleanup(func() { _ = r.Close() })
	c, cancel := ctx()
	defer cancel()
	if err := r.Ping(c).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	if err := r.FlushDB(c).Err(); err != nil {
		t.Fatal(err)
	}
	pg := testenv.NewPostgresPool(t)
	if _, err := pg.Exec(c, "TRUNCATE TABLE jobs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	logger := slog.Default()
	jm, err := job.NewJobModule(job.JobDeps{PGPool: pg, UserPolicyReader: users{}}, jobConfig.JobConfig{Tiers: map[string]jobConfig.RetryConfig{"free": {MaxAttempt: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	sm, err := scheduler.NewSchedulerModule(scheduler.SchedulerDeps{RedisClient: r, LeaseTimeout: time.Hour}, schedulerConfig.SchedulerConfig{QueueCapacity: 20, IdempotencyKeyTTL: time.Minute}, logger)
	if err != nil {
		t.Fatal(err)
	}
	smStream, err := stream.NewStreamModule(stream.StreamDeps{RedisClient: r, TokenValidationService: tokens{}}, stream.StreamConfig{}, logger)
	if err != nil {
		t.Fatal(err)
	}
	ide := idempotency.NewRedisStore(r, inferenceConfig.InferenceConfig{CompletedIdempotencyTTL: time.Hour})
	ja := recoveryAdapters.NewJobAdapter(jm.JobReader, jm.JobCreator)
	queueAdapter := recoveryAdapters.NewSchedulerAdapter(sm.RecoveryService)
	idempotencyAdapter := recoveryAdapters.NewIdempotencyStatusAdapter(ide)
	eventsAdapter := recoveryAdapters.NewEventsAdapter(smStream.Publisher())
	service := recovery.NewRecoveryService(queueAdapter, ja, ja, ja, idempotencyAdapter, eventsAdapter, time.Second, 100, time.Second, 20, logger)
	serviceB := recovery.NewRecoveryService(queueAdapter, ja, ja, ja, idempotencyAdapter, eventsAdapter, time.Second, 100, time.Second, 20, logger)
	t.Cleanup(pg.Close)
	return &harness{r: r, pg: pg, jobs: jm.JobCreator, reader: jm.JobReader, queue: sm.QueueService, claimer: sm.JobClaimerService, service: service, serviceB: serviceB, idem: ide}
}

func user(t *testing.T, h *harness) string {
	t.Helper()
	id := uuid.NewString()
	c, cancel := ctx()
	defer cancel()
	if _, err := h.pg.Exec(c, "INSERT INTO users (user_id) VALUES ($1)", id); err != nil {
		t.Fatal(err)
	}
	return id
}
func newJob(t *testing.T, h *harness, uid string) string {
	t.Helper()
	c, cancel := ctx()
	defer cancel()
	id, err := h.jobs.Create(c, jobPublic.CreateJobRequest{UserID: uid, Prompt: "recovery", Model: "model", MaxOutputTokens: 1})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func snapshot(t *testing.T, h *harness, id string) jobPublic.JobSnapshot {
	c, cancel := ctx()
	defer cancel()
	v, err := h.reader.GetByID(c, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func age(t *testing.T, h *harness, id string, at time.Time) {
	c, cancel := ctx()
	defer cancel()
	if _, err := h.pg.Exec(c, "UPDATE jobs SET created_at=$2 WHERE job_id=$1", id, at); err != nil {
		t.Fatal(err)
	}
}
func retry(t *testing.T, h *harness, id string, n int) {
	c, cancel := ctx()
	defer cancel()
	if _, err := h.pg.Exec(c, "UPDATE jobs SET retry_counts=$2 WHERE job_id=$1", id, n); err != nil {
		t.Fatal(err)
	}
}
func enqueue(t *testing.T, h *harness, id, uid string) {
	c, cancel := ctx()
	defer cancel()
	if _, err := h.queue.Enqueue(c, schedulerPublic.QueueEntry{JobID: id, UserID: uid, CurrentAttempt: 1}); err != nil {
		t.Fatal(err)
	}
}
func crashClaim(t *testing.T, h *harness, id, uid string) {
	enqueue(t, h, id, uid)
	c, cancel := ctx()
	defer cancel()
	claim, err := h.claimer.NextJob(c, 99)
	if err != nil || claim == nil || claim.JobID != id {
		t.Fatalf("worker claim=%+v err=%v", claim, err)
	}
}
func expire(t *testing.T, h *harness, id string) {
	c, cancel := ctx()
	defer cancel()
	if err := h.r.ZAdd(c, processingKey, redis.Z{Member: id, Score: 1}).Err(); err != nil {
		t.Fatal(err)
	}
}
func processing(t *testing.T, h *harness, id string) bool {
	c, cancel := ctx()
	defer cancel()
	return h.r.ZScore(c, processingKey, id).Err() == nil
}
func queue(t *testing.T, h *harness, uid string) []string {
	c, cancel := ctx()
	defer cancel()
	v, err := h.r.LRange(c, queuePrefix+uid, 0, -1).Result()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func occurrences(v []string, id string) int {
	n := 0
	for _, x := range v {
		if x == id {
			n++
		}
	}
	return n
}
func marker(t *testing.T, h *harness, id string) bool {
	c, cancel := ctx()
	defer cancel()
	v, err := h.r.SIsMember(c, completedKey, id).Result()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func seedIdem(t *testing.T, h *harness, uid, id string) {
	c, cancel := ctx()
	defer cancel()
	key := "recovery-" + id
	if _, err := h.idem.ClaimOrGet(c, uid, key, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := h.idem.SetJobID(c, uid, key, id); err != nil {
		t.Fatal(err)
	}
}
func terminalEvent(t *testing.T, h *harness, id, want string) int {
	c, cancel := ctx()
	defer cancel()
	rows, err := h.r.XRange(c, "stream:"+id, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, row := range rows {
		raw, ok := row.Values["payload"]
		if !ok {
			continue
		}
		var e struct {
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(raw.(string)), &e) == nil && e.Type == want {
			n++
		}
	}
	return n
}
func sweep(t *testing.T, h *harness, f func(context.Context) error) {
	c, cancel := ctx()
	defer cancel()
	if err := f(c); err != nil {
		t.Fatal(err)
	}
}

func TestR01ExpiredLeaseRequeuesAndRetries(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	crashClaim(t, h, id, u)
	expire(t, h, id)
	sweep(t, h, h.service.Sweep)
	if s := snapshot(t, h, id); s.RetryCount != 2 || s.Status != "created" {
		t.Fatalf("snapshot=%+v", s)
	}
	if occurrences(queue(t, h, u), id) != 1 || processing(t, h, id) {
		t.Fatalf("queue/processing invariant failed")
	}
}
func TestR02NonExpiredLeaseUntouched(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	crashClaim(t, h, id, u)
	before := snapshot(t, h, id)
	sweep(t, h, h.service.Sweep)
	after := snapshot(t, h, id)
	if after.RetryCount != before.RetryCount || len(queue(t, h, u)) != 0 || !processing(t, h, id) {
		t.Fatalf("lease changed: before=%+v after=%+v", before, after)
	}
}
func TestR03RetryExhaustionFails(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	retry(t, h, id, 3)
	seedIdem(t, h, u, id)
	crashClaim(t, h, id, u)
	expire(t, h, id)
	sweep(t, h, h.service.Sweep)
	s := snapshot(t, h, id)
	if s.Status != "failed" || len(queue(t, h, u)) != 0 || processing(t, h, id) {
		t.Fatalf("terminal recovery failed: %+v queue=%v", s, queue(t, h, u))
	}
	if terminalEvent(t, h, id, "failed") != 1 {
		t.Fatal("expected exactly one failed event")
	}
}
func TestR04CompletionMarkerReconciles(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	seedIdem(t, h, u, id)
	c, cancel := ctx()
	defer cancel()
	if err := h.r.SAdd(c, completedKey, id).Err(); err != nil {
		t.Fatal(err)
	}
	sweep(t, h, h.service.ReconcileCompleted)
	if s := snapshot(t, h, id); s.Status != "completed" {
		t.Fatalf("status=%+v", s)
	}
	if marker(t, h, id) || terminalEvent(t, h, id, "completed") != 1 {
		t.Fatal("completion marker/stream invariant failed")
	}
}
func TestR05CompletionReconciliationIdempotent(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	seedIdem(t, h, u, id)
	c, cancel := ctx()
	if err := h.r.SAdd(c, completedKey, id).Err(); err != nil {
		t.Fatal(err)
	}
	cancel()
	sweep(t, h, h.service.ReconcileCompleted)
	sweep(t, h, h.service.ReconcileCompleted)
	if s := snapshot(t, h, id); s.Status != "completed" || marker(t, h, id) || terminalEvent(t, h, id, "completed") != 1 {
		t.Fatal("reconciliation was not idempotent")
	}
}
func TestR06CompletionBeatsOrphanSweep(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now().Add(-2*time.Second))
	seedIdem(t, h, u, id)
	c, cancel := ctx()
	if err := h.r.SAdd(c, completedKey, id).Err(); err != nil {
		t.Fatal(err)
	}
	cancel()
	sweep(t, h, h.service.ReconcileCompleted)
	sweep(t, h, h.service.SweepOrphans)
	if snapshot(t, h, id).Status != "completed" || slices.Contains(queue(t, h, u), id) {
		t.Fatal("completion lost to orphan sweep")
	}
}
func TestR07GenuineOrphanRequeued(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now().Add(-2*time.Second))
	sweep(t, h, h.service.SweepOrphans)
	if occurrences(queue(t, h, u), id) != 1 {
		t.Fatal("orphan was not enqueued exactly once")
	}
}
func TestR08RecentJobNotSwept(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now())
	sweep(t, h, h.service.SweepOrphans)
	if slices.Contains(queue(t, h, u), id) {
		t.Fatal("recent job swept")
	}
}
func TestR09CompletionMarkerPreventsEnqueue(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now().Add(-2*time.Second))
	c, cancel := ctx()
	if err := h.r.SAdd(c, completedKey, id).Err(); err != nil {
		t.Fatal(err)
	}
	cancel()
	sweep(t, h, h.service.SweepOrphans)
	if slices.Contains(queue(t, h, u), id) {
		t.Fatal("marked job enqueued")
	}
}
func TestR10ProcessingJobNotSwept(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now().Add(-2*time.Second))
	crashClaim(t, h, id, u)
	sweep(t, h, h.service.SweepOrphans)
	if slices.Contains(queue(t, h, u), id) {
		t.Fatal("processing job swept")
	}
}
func TestR11AlreadyQueuedNotDuplicated(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	age(t, h, id, time.Now().Add(-2*time.Second))
	enqueue(t, h, id, u)
	sweep(t, h, h.service.SweepOrphans)
	if occurrences(queue(t, h, u), id) != 1 {
		t.Fatalf("queue=%v", queue(t, h, u))
	}
}
func TestR14ConcurrentSweepsRecoverOnce(t *testing.T) {
	h := newHarness(t)
	u := user(t, h)
	id := newJob(t, h, u)
	crashClaim(t, h, id, u)
	expire(t, h, id)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	for _, service := range []*recovery.RecoveryService{h.service, h.serviceB} {
		go func(s *recovery.RecoveryService) {
			defer wg.Done()
			c, cancel := ctx()
			defer cancel()
			errs <- s.Sweep(c)
		}(service)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if s := snapshot(t, h, id); s.RetryCount != 2 || occurrences(queue(t, h, u), id) != 1 || processing(t, h, id) {
		t.Fatalf("concurrent recovery invariant failed: %+v queue=%v", s, queue(t, h, u))
	}
}

var _ streamPorts.AccessTokenValidator = tokens{}
