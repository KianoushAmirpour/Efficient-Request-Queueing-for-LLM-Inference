package transport

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"efficient-request-queueing-for-llm-inference/internal/auth/usecases"
)

type AuthHandler struct {
	authUseCase *usecases.AuthUseCase
	logger      *slog.Logger
}

func NewAuthHandler(
	authUseCase *usecases.AuthUseCase,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
		logger:      logger,
	}
}

func (h *AuthHandler) HandleOauthRedirectGithub(c *gin.Context) {

	authURL, err := h.authUseCase.GenerateAuthURL(c.Request.Context())

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) HandleOauthCallback(c *gin.Context) {

	var req OauthCallbackRequest

	err := c.ShouldBindQuery(&req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	code := req.Code
	state := req.State

	authOutput, err := h.authUseCase.HandleCallback(c.Request.Context(), code, state)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Header("Authorization", "Bearer "+authOutput.AccessToken)
	c.JSON(
		http.StatusOK,
		OauthCallbackResponse{
			Message:     "Authentication successful",
			AccessToken: authOutput.AccessToken,
		},
	)
}
