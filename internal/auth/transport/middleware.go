package transport

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	publicLoggingAPI "efficient-request-queueing-for-llm-inference/internal/observability/logger"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

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

		resolved := sharederr.ResolveError(ginErr.Err, AuthErrCodeRegistry, "OAUTH_FAILED")

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
