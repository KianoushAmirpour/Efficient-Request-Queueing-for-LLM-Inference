package adapters

import (
	"context"
	"fmt"

	schedulerPublicAPI "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type JobClaimerAdapter struct {
	claimer schedulerPublicAPI.JobClaimer
}

var _ domain.JobClaimer = JobClaimerAdapter{}

func NewJobClaimerAdapter(claimer schedulerPublicAPI.JobClaimer) JobClaimerAdapter {
	return JobClaimerAdapter{
		claimer: claimer,
	}
}

func (a JobClaimerAdapter) ClaimNextJob(ctx context.Context, workerID int) (*domain.JobClaimResult, error) {
	result, err := a.claimer.NextJob(ctx, workerID)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return &domain.JobClaimResult{
		JobID:  result.JobID,
		UserID: result.UserID,
	}, nil
}

type QueueAdapter struct {
	queueService schedulerPublicAPI.QueueService
}

var _ domain.Queue = QueueAdapter{}

func NewQueueAdapter(queueService schedulerPublicAPI.QueueService) QueueAdapter {
	return QueueAdapter{
		queueService: queueService,
	}
}

func (a QueueAdapter) Release(ctx context.Context, jobID string) error {
	return a.queueService.ReleaseProcessingJob(ctx, jobID)
}

func (a QueueAdapter) ExtendLease(ctx context.Context, jobID string) error {
	err := a.queueService.ExtendProcessingLease(ctx, jobID)
	if sharederr.IsCode(err, schedulerPublicAPI.ErrCodeJobNotFound) {
		return fmt.Errorf("%w: %w", domain.ErrLeaseExtendFailed, err)
	}
	return err
}

func (a QueueAdapter) MarkCompleted(ctx context.Context, jobID string) error {
	return a.queueService.MarkJobCompleted(ctx, jobID)
}
