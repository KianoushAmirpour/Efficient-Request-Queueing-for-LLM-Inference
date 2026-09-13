package ports

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type Enqueuer interface {
	TryEnqueue(ctx context.Context, job domain.InferenceJob) (becomeActive bool, err error)
}
