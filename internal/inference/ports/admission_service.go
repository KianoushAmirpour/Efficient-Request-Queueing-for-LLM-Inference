package ports

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type RequestAdmitter interface {
	Admit(ctx context.Context, inferenceReq *domain.InferenceRequest) (*domain.InferenceRequest, error)
}
