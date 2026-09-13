package adapters

import (
	"context"
	"time"

	jobPublic "efficient-request-queueing-for-llm-inference/internal/job/public"
	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type JobAdapter struct {
	reader  jobPublic.JobReader
	service jobPublic.JobService
}

var (
	_ domain.Jobs         = JobAdapter{}
	_ domain.StatusWriter = JobAdapter{}
	_ domain.Policies     = JobAdapter{}
)

func NewJobAdapter(reader jobPublic.JobReader, service jobPublic.JobService) JobAdapter {
	return JobAdapter{reader: reader, service: service}
}

func (a JobAdapter) ByID(ctx context.Context, jobID string) (domain.Job, error) {
	job, err := a.reader.GetByID(ctx, jobID)
	if err != nil {
		if sharederr.IsCode(err, jobPublic.ErrCodeJobNotFound) {
			return domain.Job{}, domain.ErrJobNotFound
		}
		return domain.Job{}, err
	}
	return domain.Job{JobID: job.JobID, UserID: job.UserID, Status: job.Status, RetryCount: job.RetryCount}, nil
}

func (a JobAdapter) MarkFailed(ctx context.Context, jobID string, retryCount int) error {
	return a.service.MarkFailed(ctx, jobID, retryCount)
}

func (a JobAdapter) UpdateCreated(ctx context.Context, jobID string, retryCount int) error {
	return a.service.UpdateStatus(ctx, jobID, domain.StatusCreated, retryCount)
}

func (a JobAdapter) MarkCompleted(ctx context.Context, jobID string, retryCount int) error {
	return a.service.MarkCompleted(ctx, jobID, retryCount)
}

func (a JobAdapter) PendingOrphans(ctx context.Context, before time.Time, limit int) ([]domain.OrphanJob, error) {
	jobs, err := a.reader.PendingOrphans(ctx, before, limit)
	if err != nil {
		return nil, err
	}
	result := make([]domain.OrphanJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, domain.OrphanJob{Job: domain.Job{JobID: job.JobID, UserID: job.UserID, Status: job.Status, RetryCount: job.RetryCount}, CreatedAt: job.CreatedAt})
	}
	return result, nil
}

func (a JobAdapter) GetRetryPolicy(ctx context.Context, userID string) (domain.RetryPolicy, error) {
	policy, err := a.service.GetRetryPolicy(ctx, userID)
	if err != nil {
		return domain.RetryPolicy{}, err
	}
	return domain.RetryPolicy{MaxAttempts: policy.MaxAttempts}, nil
}
