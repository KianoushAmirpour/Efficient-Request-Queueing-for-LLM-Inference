package transport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	publicLoggingAPI "efficient-request-queueing-for-llm-inference/internal/observability/logger"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/user/domain"
)

func AdminAuthMiddleware(tokenService domain.AuthTokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeUnauthorized,
				ErrType: "INVALID_AUTH_HEADER",
				Err:     fmt.Errorf("authorization header must be in the format 'Bearer <token>'"),
			})
			c.Abort()
			return
		}

		token := parts[1]
		adminID, role, err := tokenService.ValidateAccessToken(token)
		if err != nil {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeUnauthorized,
				ErrType: "INVALID_ACCESS_TOKEN",
				Err:     err})
			c.Abort()
			return
		}

		if role != string(domain.RoleAdmin) || role == "" {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeForbidden,
				ErrType: "FORBIDDEN",
				Err:     fmt.Errorf("user %q with role %q is not authorized to access admin endpoints", adminID, role)})
			c.Abort()
			return
		}

		c.Set("user_id", adminID)
		c.Set("user_role", role)

		ctx := publicLoggingAPI.WithUserID(c.Request.Context(), adminID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func ErrorHandler(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		ginErr := c.Errors.Last()

		resolved := sharederr.ResolveError(ginErr.Err, UserCodeRegistry, "USER_REQUEST_FAILED")

		log.ErrorContext(
			c.Request.Context(),
			"http request failed",
			"http.method", c.Request.Method,
			"http.path", c.Request.URL.Path,
			"http.status_code", resolved.StatusCode,
			"error.code", resolved.ErrorCode,
			"error.type", resolved.ErrorType,
			"error", resolved.Error,
		)

		var resp sharederr.ErrorResponse

		resp.Error.ErrorCode = resolved.ErrorCode
		resp.Error.Message = resolved.Message
		resp.Error.RequestID = publicLoggingAPI.RequestID(c.Request.Context())

		c.AbortWithStatusJSON(resolved.StatusCode, resp)
	}
}
