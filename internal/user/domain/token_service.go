package domain

type AuthTokenManager interface {
	Generate(userID string, role string) (string, error)

	ValidateAccessToken(token string) (userID string, role string, err error)
}
