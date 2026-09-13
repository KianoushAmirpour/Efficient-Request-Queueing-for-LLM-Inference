package adapters

import (
	authPublicAPI "efficient-request-queueing-for-llm-inference/internal/auth/public"
	"efficient-request-queueing-for-llm-inference/internal/stream/ports"
)

type TokenAuthenticatorAdapter struct {
	TokenManager authPublicAPI.AuthTokenManager
}

var _ ports.AccessTokenValidator = TokenAuthenticatorAdapter{}

func NewTokenAuthenticatorAdapter(tokenManager authPublicAPI.AuthTokenManager) TokenAuthenticatorAdapter {
	return TokenAuthenticatorAdapter{
		TokenManager: tokenManager,
	}
}

func (a TokenAuthenticatorAdapter) ValidateAccessToken(tokenString string) (string, string, error) {
	claims, err := a.TokenManager.ValidateAccessToken(tokenString)
	if err != nil {
		return "", "", err
	}
	return claims.UserID, claims.Role, nil
}
