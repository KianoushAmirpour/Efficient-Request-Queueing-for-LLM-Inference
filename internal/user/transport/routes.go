package transport

import (
	"time"

	"github.com/gin-gonic/gin"

	"efficient-request-queueing-for-llm-inference/internal/user/domain"
)

func RegisterRoutes(rg *gin.RouterGroup, handler *AdminHandler, tokenService domain.AuthTokenManager) {
	admin := rg.Group("/admin")
	admin.Use(TimeoutMiddleware(5*time.Second), ErrorHandler(handler.logger))
	admin.POST("/register", handler.HandleCreateAdmin)
	admin.POST("/login", handler.HandleAdminLogin)
	adminAuth := admin.Group("")
	adminAuth.Use(AdminAuthMiddleware(tokenService))
	{
		adminAuth.PUT("/users/:user_id/tier", handler.HandleUpdateUserTier)
	}
}
