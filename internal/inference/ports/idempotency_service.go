package ports

import (
	"context"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type IdempotencyStore interface {
	ClaimOrGet(ctx context.Context, userID string, idempotencyKey string, ttl time.Duration) (*domain.IdempotencyResult, error)
	SetJobID(ctx context.Context, userID, idempotencyHeader, jobID string) error
	Delete(ctx context.Context, userID, idempotencyKey string) error
	TransitionStatus(ctx context.Context, userID, jobID, status string) error
}
