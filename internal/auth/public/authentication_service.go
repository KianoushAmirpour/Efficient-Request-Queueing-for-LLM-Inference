package public

type Claims struct {
	UserID string
	Role   string
}

type AuthTokenManager interface {
	Generate(user Claims) (string, error)
	ValidateAccessToken(tokenString string) (Claims, error)
}
