package transport

type OauthCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

type OauthCallbackResponse struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}
