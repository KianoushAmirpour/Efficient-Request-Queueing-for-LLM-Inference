package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	inferencePublic "efficient-request-queueing-for-llm-inference/internal/inference/public"
	publicLoggingAPI "efficient-request-queueing-for-llm-inference/internal/observability/logger"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

func AuthenticateMiddleware(
	log *slog.Logger,
	validator ports.AccessTokenValidator,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeUnauthorized,
				ErrType: "INVALID_AUTH_HEADER",
				Err: fmt.Errorf(
					"authorization header must be in the format 'Bearer <token>'",
				),
			})

			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeUnauthorized,
				ErrType: "INVALID_AUTH_HEADER",
				Err:     fmt.Errorf("bearer token is empty"),
			})

			c.Abort()
			return
		}

		userID, _, err := validator.ValidateAccessToken(tokenString)
		if err != nil {
			_ = c.Error(&sharederr.AppError{
				ErrCode: sharederr.ErrCodeUnauthorized,
				ErrType: "INVALID_ACCESS_TOKEN",
				Err:     err,
			})

			c.Abort()
			return
		}

		c.Set("userID", userID)

		ctx = publicLoggingAPI.WithUserID(ctx, userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func IdempotencyKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyHeader := c.GetHeader("Idempotency-Key")
		if idempotencyHeader == "" {
			_ = c.Error(&sharederr.AppError{
				ErrCode: inferencePublic.ErrCodeIdempotencyKeyMissing,
				ErrType: "MISSING_IDEMPOTENCY_KEY",
				Err:     fmt.Errorf("Idempotency-Key header is required for this endpoint")})
			c.Abort()
			return
		}

		c.Set("Idempotency-Header", idempotencyHeader)

		c.Next()
	}
}

func MaxBodySizeMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			limit,
		)

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

		resolved := sharederr.ResolveError(ginErr.Err, InferenceCodeRegistry, "INFERENCE_FAILED")

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
