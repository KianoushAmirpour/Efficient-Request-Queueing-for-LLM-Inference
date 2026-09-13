package usecase

import (
	"context"
	"errors"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/scheduler/domain"
	"efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type QueueService struct {
	jobQueue domain.JobQueue
}

func NewQueueService(jobQueue domain.JobQueue) public.QueueService {
	return QueueService{
		jobQueue: jobQueue,
	}
}

func (q QueueService) Enqueue(ctx context.Context, entry public.QueueEntry) (public.EnqueueResult, error) {

	req := domain.QueueEntry{
		JobID:          entry.JobID,
		UserID:         entry.UserID,
		CurrentAttempt: entry.CurrentAttempt,
		EnqueuedAt:     time.Now(),
	}

	result, err := q.jobQueue.PushLeft(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrQueueFull):
			return public.EnqueueResult{}, sharederr.EnsureAppError(err, public.ErrCodeQueueFull, ErrTypeScheduler)
		case errors.Is(err, domain.ErrDuplicateJob):
			return public.EnqueueResult{}, sharederr.EnsureAppError(err, public.ErrCodeDuplicatedJob, ErrTypeScheduler)
		default:
			return public.EnqueueResult{}, sharederr.EnsureAppError(err, public.ErrCodeEnqueueFailed, ErrTypeScheduler)
		}
	}

	return public.EnqueueResult{
		BecameActive: result.BecameActive,
	}, nil
}

func (q QueueService) ReleaseProcessingJob(ctx context.Context, jobID string) error {
	if err := q.jobQueue.ReleaseProcessingJob(ctx, jobID); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeReleaseProcessingFailed, ErrTypeScheduler)
	}
	return nil
}

func (q QueueService) ExtendProcessingLease(ctx context.Context, jobID string) error {
	if err := q.jobQueue.ExtendProcessingLease(ctx, jobID); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeReleaseExtendingFailed, ErrTypeScheduler)
	}
	return nil
}

func (q QueueService) MarkJobCompleted(ctx context.Context, jobID string) error {
	return q.jobQueue.MarkJobCompleted(ctx, jobID)
}
