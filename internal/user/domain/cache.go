package domain

import "context"

type UserTierCache interface {
	Invalidate(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (string, error)
	Set(ctx context.Context, userID, userTier string) error
}
