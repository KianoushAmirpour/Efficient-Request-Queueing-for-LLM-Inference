package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	publicLoggingAPI "efficient-request-queueing-for-llm-inference/internal/observability/logger"
)

const requestIDHeader = "X-Request-Id"

func RequestContextMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := publicLoggingAPI.WithRequestID(
			c.Request.Context(),
			requestID,
		)

		c.Request = c.Request.WithContext(ctx)

		c.Writer.Header().Set(requestIDHeader, requestID)

		c.Next()

		log.InfoContext(
			ctx,
			"HTTP request completed",
			"http.method", c.Request.Method,
			"http.path", c.Request.URL.Path,
			"http.ip", c.ClientIP(),
			"http.status_code", c.Writer.Status(),
			"http.duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

func RecoveryMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx := c.Request.Context()

				log.ErrorContext(
					ctx,
					"panic recovered",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"http.method", c.Request.Method,
					"http.path", c.Request.URL.Path,
				)

				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}
