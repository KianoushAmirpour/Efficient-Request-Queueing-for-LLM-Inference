package ports

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type JobService interface {
	Create(ctx context.Context, task *domain.InferenceRequest) (*domain.InferenceJob, error)
	MarkFailed(ctx context.Context, jobID string, retryCount int) error
}
