package domain

import (
	"context"
)

type OAuthAuthorizer interface {
	AuthorizationURL(state string) string
	ExchangeCode(ctx context.Context, code string) (string, error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUser, error)
}
