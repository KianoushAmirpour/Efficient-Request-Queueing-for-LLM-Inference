package domain

import (
	"context"
)

type AdminService interface {
	SetUserAsAdmin(ctx context.Context, userID string) error

	UpdateUserTier(ctx context.Context, userID string, tier string) error

	IsUserAdmin(ctx context.Context, userID string) error
}
