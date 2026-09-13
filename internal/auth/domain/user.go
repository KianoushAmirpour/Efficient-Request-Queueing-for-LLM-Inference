package domain

type OAuthUser struct {
	Provider       string
	ProviderUserID string
}

type AuthenticatedUser struct {
	UserID string
	Role   string
}

type AuthResponse struct {
	AccessToken string
}
