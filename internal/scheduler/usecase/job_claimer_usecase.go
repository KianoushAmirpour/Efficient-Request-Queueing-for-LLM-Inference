package usecase

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/scheduler/domain"
	"efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeScheduler = "SCHEDULER_FAILED"

type JobClaimerUseCase struct {
	NextJobClaimerService domain.JobQueue
}

func NewJobClaimerUseCase(nextJobClaimer domain.JobQueue) public.JobClaimer {
	return &JobClaimerUseCase{
		NextJobClaimerService: nextJobClaimer,
	}
}

func (d *JobClaimerUseCase) NextJob(ctx context.Context, workerID int) (*public.NextJobClaim, error) {
	dequeueResult, err := d.NextJobClaimerService.ClaimNextJob(ctx, workerID)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, public.ErrCodeJobClaimerFailed, ErrTypeScheduler)
	}

	if dequeueResult == nil {
		return nil, nil
	}

	return &public.NextJobClaim{JobID: dequeueResult.JobID, UserID: dequeueResult.UserID}, nil
}
