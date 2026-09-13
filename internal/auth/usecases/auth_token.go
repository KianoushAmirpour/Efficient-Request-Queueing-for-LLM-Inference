package usecases

import (
	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	authPublic "efficient-request-queueing-for-llm-inference/internal/auth/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type Authenticator struct {
	authTokenManager domain.AuthTokenManager
}

func NewAuthenticator(authManager domain.AuthTokenManager) Authenticator {
	return Authenticator{authTokenManager: authManager}
}

func (a Authenticator) Generate(sessionUser authPublic.Claims) (string, error) {

	claims := domain.AuthenticatedUser{
		UserID: sessionUser.UserID,
		Role:   sessionUser.Role,
	}

	token, err := a.authTokenManager.Generate(&claims)
	if err != nil {
		return "", sharederr.EnsureAppError(err, authPublic.ErrCodeJWTTokenGenerationFailed, ErrTypeOAuth)
	}

	return token, nil
}

func (a Authenticator) ValidateAccessToken(tokenString string) (authPublic.Claims, error) {
	userID, role, err := a.authTokenManager.ValidateAccessToken(tokenString)
	if err != nil {
		return authPublic.Claims{}, sharederr.EnsureAppError(err, authPublic.ErrCodeJWTTokenValidationFailed, ErrTypeOAuth)
	}
	return authPublic.Claims{UserID: userID, Role: role}, nil
}
