package concurrency_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	job "efficient-request-queueing-for-llm-inference/internal/job"
	jobDomain "efficient-request-queueing-for-llm-inference/internal/job/domain"
	jobConfig "efficient-request-queueing-for-llm-inference/internal/job/infrastructure/config"
	jobPublic "efficient-request-queueing-for-llm-inference/internal/job/public"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

type jobConcurrencyUsers struct{}

func (jobConcurrencyUsers) GetUserPolicy(context.Context, string) (jobDomain.UserPolicy, error) {
	return jobDomain.UserPolicy{Tier: "free"}, nil
}

func TestConcurrentConditionalUpdateAllowsOneClaim(t *testing.T) {
	pg := testenv.NewPostgresPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := pg.Exec(ctx, "TRUNCATE TABLE jobs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("clear job tables: %v", err)
	}
	userID := uuid.NewString()
	if _, err := pg.Exec(ctx, "INSERT INTO users (user_id) VALUES ($1)", userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	module, err := job.NewJobModule(job.JobDeps{PGPool: pg, UserPolicyReader: jobConcurrencyUsers{}}, jobConfig.JobConfig{
		Tiers: map[string]jobConfig.RetryConfig{"free": {MaxAttempt: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second}},
	}, slog.Default())
	if err != nil {
		t.Fatalf("construct job module: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	jobID, err := module.JobCreator.Create(ctx, jobPublic.CreateJobRequest{UserID: userID, Prompt: "claim me", Model: "concurrency-model", MaxOutputTokens: 2})
	cancel()
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, retryCount := range []int{2, 3} {
		wg.Add(1)
		go func(retryCount int) {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			updated, err := module.JobCreator.UpdateStatusIfStatus(ctx, jobID, retryCount, string(jobDomain.StatusCompleted), string(jobDomain.StatusCreated))
			results <- updated
			errs <- err
		}(retryCount)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	updatedCount := 0
	for updated := range results {
		if updated {
			updatedCount++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("conditional update error: %v", err)
		}
	}
	if updatedCount != 1 {
		t.Fatalf("successful conditional updates=%d; want exactly one", updatedCount)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	var status string
	var retryCount int
	if err := pg.QueryRow(ctx, "SELECT status, retry_counts FROM jobs WHERE job_id = $1", jobID).Scan(&status, &retryCount); err != nil {
		t.Fatalf("read final job: %v", err)
	}
	cancel()
	if status != string(jobDomain.StatusCompleted) || (retryCount != 2 && retryCount != 3) {
		t.Fatalf("final status=%q retry count=%d; want completed with one winning retry count", status, retryCount)
	}
}
