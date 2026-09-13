package ports

type AccessTokenValidator interface {
	ValidateAccessToken(tokenString string) (userID string, role string, err error)
}
