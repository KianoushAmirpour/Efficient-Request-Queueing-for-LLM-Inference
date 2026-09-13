package transport

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"efficient-request-queueing-for-llm-inference/internal/user/domain"
	"efficient-request-queueing-for-llm-inference/internal/user/usecase"
)

type AdminHandler struct {
	adminUseCase *usecase.AdminUseCase
	logger       *slog.Logger
}

func NewAdminHandler(adminUseCase *usecase.AdminUseCase, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		adminUseCase: adminUseCase,
		logger:       logger,
	}
}

func (h *AdminHandler) HandleCreateAdmin(c *gin.Context) {
	ctx := c.Request.Context()

	response, err := h.adminUseCase.CreateAdmin(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, CreateAdminResponse{
		Message: "Admin user created successfully",
		UserID:  response.UserID,
	})
}

func (h *AdminHandler) HandleAdminLogin(c *gin.Context) {
	ctx := c.Request.Context()

	var req AdminAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	response, err := h.adminUseCase.AdminAuthLogin(ctx, req.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, AdminAuthLoginResponse{
		Message:     "Authentication successful",
		AccessToken: response.Token,
	})
}

func (h *AdminHandler) HandleUpdateUserTier(c *gin.Context) {
	ctx := c.Request.Context()

	userID := c.Param("user_id")

	var req UpdateUserTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	updateReq := &domain.UpdateUserTierRequest{
		UserID: userID,
		Tier:   req.Tier,
	}

	if err := h.adminUseCase.UpdateUserTier(ctx, updateReq); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK,
		UpdateUserTierResponse{
			Message: fmt.Sprintf("Your plan successfully changed to %s", req.Tier),
			Tier:    req.Tier,
		},
	)
}
