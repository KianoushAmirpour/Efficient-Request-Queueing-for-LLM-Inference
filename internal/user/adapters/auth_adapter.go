package adapters

import (
	authPublicAPI "efficient-request-queueing-for-llm-inference/internal/auth/public"
	"efficient-request-queueing-for-llm-inference/internal/user/domain"
)

type TokenAuthenticatorAdapter struct {
	TokenManager authPublicAPI.AuthTokenManager
}

var _ domain.AuthTokenManager = TokenAuthenticatorAdapter{}

func NewTokenAuthenticatorAdapter(tokenManager authPublicAPI.AuthTokenManager) TokenAuthenticatorAdapter {
	return TokenAuthenticatorAdapter{
		TokenManager: tokenManager,
	}
}

func (a TokenAuthenticatorAdapter) Generate(userID string, role string) (string, error) {
	claims := authPublicAPI.Claims{
		UserID: userID,
		Role:   role,
	}

	token, err := a.TokenManager.Generate(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (a TokenAuthenticatorAdapter) ValidateAccessToken(tokenString string) (string, string, error) {
	authenticatedSession, err := a.TokenManager.ValidateAccessToken(tokenString)
	if err != nil {
		return "", "", err
	}
	return authenticatedSession.UserID, authenticatedSession.Role, nil
}
