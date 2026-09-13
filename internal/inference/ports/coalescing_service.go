package ports

import (
	"context"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type Coalescer interface {
	TryCoalesce(ctx context.Context, userID string, hash uint64, ttl time.Duration) (domain.CoalescingDecision, error)
	Delete(ctx context.Context, userID string, hashRequest uint64) error
}

type Hasher interface {
	HashRequest(key string) uint64
}
