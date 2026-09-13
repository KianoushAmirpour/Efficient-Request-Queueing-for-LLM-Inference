package domain

import (
	"context"
	"time"
)

type OAuthStateGenerator interface {
	Generate() (string, error)
}

type OAuthStateStore interface {
	Store(ctx context.Context, state, provider string, ttl time.Duration) error
	Exists(ctx context.Context, state, provider string) error
	Delete(ctx context.Context, state, provider string) error
}
