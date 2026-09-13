package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	schedulerPublicAPI "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type JobEnqueuerAdapter struct {
	queueService schedulerPublicAPI.QueueService
}

var _ ports.Enqueuer = JobEnqueuerAdapter{}

func NewJobEnqueuerAdapter(queueService schedulerPublicAPI.QueueService) JobEnqueuerAdapter {
	return JobEnqueuerAdapter{
		queueService: queueService,
	}
}

func (a JobEnqueuerAdapter) TryEnqueue(ctx context.Context, job domain.InferenceJob) (becomeActive bool, err error) {

	enqueueResult, err := a.queueService.Enqueue(
		ctx,
		schedulerPublicAPI.QueueEntry{
			JobID:          job.JobID,
			UserID:         job.UserID,
			CurrentAttempt: job.CurrentAttempt,
		})
	if err != nil {
		if sharederr.IsCode(err, schedulerPublicAPI.ErrCodeQueueFull) {
			return false, domain.ErrQueueFull
		}
		return false, err
	}
	if !enqueueResult.BecameActive {
		return false, nil
	}
	return true, nil

}
