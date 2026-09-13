package transport

type UpdateUserTierRequest struct {
	Tier string `json:"tier" binding:"required,oneof=free premium"`
}

type CreateAdminResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

type AdminAuthLoginRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type AdminAuthLoginResponse struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}

type UpdateUserTierResponse struct {
	Message string `json:"message"`
	Tier    string `json:"tier"`
}
