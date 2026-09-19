package job_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	job "efficient-request-queueing-for-llm-inference/internal/job"
	jobDomain "efficient-request-queueing-for-llm-inference/internal/job/domain"
	jobConfig "efficient-request-queueing-for-llm-inference/internal/job/infrastructure/config"
	jobPublic "efficient-request-queueing-for-llm-inference/internal/job/public"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

type users struct{}

func (users) GetUserPolicy(context.Context, string) (jobDomain.UserPolicy, error) {
	return jobDomain.UserPolicy{Tier: "free"}, nil
}

type harness struct {
	pg      *pgxpool.Pool
	service jobPublic.JobService
	reader  jobPublic.JobReader
}

func newHarness(t *testing.T) harness {
	t.Helper()
	pg := testenv.NewPostgresPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := pg.Exec(ctx, "TRUNCATE TABLE jobs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("clear job tables: %v", err)
	}
	module, err := job.NewJobModule(job.JobDeps{PGPool: pg, UserPolicyReader: users{}}, jobConfig.JobConfig{
		Tiers: map[string]jobConfig.RetryConfig{"free": {MaxAttempt: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second}},
	}, slog.Default())
	if err != nil {
		t.Fatalf("construct job module: %v", err)
	}
	return harness{pg: pg, service: module.JobCreator, reader: module.JobReader}
}

func seedUser(t *testing.T, pg *pgxpool.Pool) string {
	t.Helper()
	userID := uuid.NewString()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := pg.Exec(ctx, "INSERT INTO users (user_id) VALUES ($1)", userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

func createJob(t *testing.T, h harness, userID string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	jobID, err := h.service.Create(ctx, jobPublic.CreateJobRequest{UserID: userID, Prompt: "integration prompt", Model: "integration-model", MaxOutputTokens: 12})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	return jobID
}

func TestCreateJobTransactionRollsBackRelatedWrites(t *testing.T) {
	h := newHarness(t)
	userID := seedUser(t, h.pg)
	tooLongModel := strings.Repeat("m", 37) // job_requests.model is VARCHAR(36).

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := h.service.Create(ctx, jobPublic.CreateJobRequest{UserID: userID, Prompt: "must roll back", Model: tooLongModel, MaxOutputTokens: 1})
	cancel()
	if err == nil {
		t.Fatal("create with invalid request payload succeeded")
	}

	var jobs, requests int
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	if err := h.pg.QueryRow(ctx, "SELECT COUNT(*) FROM jobs").Scan(&jobs); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if err := h.pg.QueryRow(ctx, "SELECT COUNT(*) FROM job_requests").Scan(&requests); err != nil {
		t.Fatalf("count job requests: %v", err)
	}
	cancel()
	if jobs != 0 || requests != 0 {
		t.Fatalf("failed creation left jobs=%d job_requests=%d; want both zero", jobs, requests)
	}
}

func TestGetExistingJobAndPayload(t *testing.T) {
	h := newHarness(t)
	userID := seedUser(t, h.pg)
	jobID := createJob(t, h, userID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	snapshot, err := h.reader.GetByID(ctx, jobID)
	cancel()
	if err != nil {
		t.Fatalf("get existing job: %v", err)
	}
	if snapshot.JobID != jobID || snapshot.UserID != userID || snapshot.Status != string(jobDomain.StatusCreated) || snapshot.RetryCount != 1 {
		t.Fatalf("snapshot=%+v; want created job with attempt 1", snapshot)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	payload, err := h.service.PayloadByID(ctx, jobPublic.JobPayloadRequest{JobID: jobID, UserID: userID})
	cancel()
	if err != nil || payload.JobID != jobID || payload.UserID != userID || payload.Prompt != "integration prompt" || payload.Model != "integration-model" || payload.MaxOutputTokens != 12 {
		t.Fatalf("payload=%+v, err=%v", payload, err)
	}
}

func TestMissingJobReturnsNotFound(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := h.reader.GetByID(ctx, uuid.NewString())
	cancel()
	if !errors.Is(err, jobDomain.ErrJobNotFound) {
		t.Fatalf("missing job error=%v; want %v", err, jobDomain.ErrJobNotFound)
	}
}

func TestValidTerminalTransition(t *testing.T) {
	h := newHarness(t)
	jobID := createJob(t, h, seedUser(t, h.pg))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := h.service.MarkCompleted(ctx, jobID, 4)
	cancel()
	if err != nil {
		t.Fatalf("complete job: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	snapshot, err := h.reader.GetByID(ctx, jobID)
	cancel()
	if err != nil || snapshot.Status != string(jobDomain.StatusCompleted) || snapshot.RetryCount != 4 {
		t.Fatalf("snapshot=%+v, err=%v; want completed with retry count 4", snapshot, err)
	}
}

func TestInvalidTerminalTransitionDoesNotChangeCompletedJob(t *testing.T) {
	h := newHarness(t)
	jobID := createJob(t, h, seedUser(t, h.pg))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := h.service.MarkCompleted(ctx, jobID, 2); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	err := h.service.MarkFailed(ctx, jobID, 3)
	cancel()
	if err == nil {
		t.Fatal("failed transition from completed succeeded")
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	snapshot, readErr := h.reader.GetByID(ctx, jobID)
	cancel()
	if readErr != nil || snapshot.Status != string(jobDomain.StatusCompleted) || snapshot.RetryCount != 2 {
		t.Fatalf("after invalid transition snapshot=%+v, err=%v", snapshot, readErr)
	}
}

func TestRetryCountUpdatePreservesCreatedStatus(t *testing.T) {
	h := newHarness(t)
	jobID := createJob(t, h, seedUser(t, h.pg))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	updated, err := h.service.UpdateStatusIfStatus(ctx, jobID, 3, string(jobDomain.StatusCreated), string(jobDomain.StatusCreated))
	cancel()
	if err != nil || !updated {
		t.Fatalf("conditional retry update=(%t, %v); want (true, nil)", updated, err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	snapshot, err := h.reader.GetByID(ctx, jobID)
	cancel()
	if err != nil || snapshot.Status != string(jobDomain.StatusCreated) || snapshot.RetryCount != 3 {
		t.Fatalf("snapshot=%+v, err=%v; want created with retry count 3", snapshot, err)
	}
}
