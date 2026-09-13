package transport

import (
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, handler *AuthHandler) {
	auth := rg.Group("/auth")
	auth.Use(TimeoutMiddleware(20*time.Second), ErrorHandler(handler.logger))

	auth.GET("/login/github", handler.HandleOauthRedirectGithub)
	auth.GET("/github/callback", handler.HandleOauthCallback)
}
