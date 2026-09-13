package domain

import (
	"context"
)

type OAuthAccountRepository interface {
	FindByProviderUserID(ctx context.Context, provider, providerUserID string) (*AuthenticatedUser, error)
	Create(ctx context.Context, providerUserID, provider, userUUID string) (*AuthenticatedUser, error)
}
