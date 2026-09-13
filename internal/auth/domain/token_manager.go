package domain

type AuthTokenManager interface {
	Generate(user *AuthenticatedUser) (string, error)
	ValidateAccessToken(tokenString string) (userID string, role string, err error)
}
