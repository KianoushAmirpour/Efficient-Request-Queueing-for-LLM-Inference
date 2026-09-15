package adapters

import (
	"context"

	inferencePublic "efficient-request-queueing-for-llm-inference/internal/inference/public"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type IdempotencyStatusAdapter struct {
	service inferencePublic.IdempotencyService
}

var _ domain.IdempotencyStatusUpdater = IdempotencyStatusAdapter{}

func NewIdempotencyStatusAdapter(service inferencePublic.IdempotencyService) IdempotencyStatusAdapter {
	return IdempotencyStatusAdapter{service: service}
}

func (a IdempotencyStatusAdapter) TransitionStatus(ctx context.Context, userID, jobID, status string) error {
	return a.service.TransitionStatus(ctx, jobID, status)
}
