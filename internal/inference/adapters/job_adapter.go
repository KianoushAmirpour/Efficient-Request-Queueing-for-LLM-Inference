package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	jobPublicAPI "efficient-request-queueing-for-llm-inference/internal/job/public"
)

type JobCreatorAdapter struct {
	jobService jobPublicAPI.JobService
}

var _ ports.JobService = JobCreatorAdapter{}

func NewJobCreatorAdapter(jobCreator jobPublicAPI.JobService) JobCreatorAdapter {
	return JobCreatorAdapter{
		jobService: jobCreator,
	}
}

func (a JobCreatorAdapter) Create(ctx context.Context, task *domain.InferenceRequest) (*domain.InferenceJob, error) {

	jobCandidate := jobPublicAPI.CreateJobRequest{
		UserID:          task.UserID,
		Prompt:          task.Prompt,
		Model:           task.Model,
		MaxOutputTokens: task.MaxOutputTokens,
	}

	jobID, err := a.jobService.Create(ctx, jobCandidate)
	if err != nil {
		return nil, err
	}

	return &domain.InferenceJob{
		JobID:  jobID,
		UserID: task.UserID,
	}, nil
}

func (a JobCreatorAdapter) MarkFailed(ctx context.Context, jobID string, retryCount int) error {
	return a.jobService.MarkFailed(ctx, jobID, retryCount)
}
