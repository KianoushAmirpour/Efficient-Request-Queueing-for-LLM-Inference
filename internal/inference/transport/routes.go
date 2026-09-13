package transport

import (
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterInferenceRoutes(rg *gin.RouterGroup, handler *SubmitInferenceHandler) {
	inference := rg.Group("/request")
	inference.Use(
		TimeoutMiddleware(10*time.Second),
		ErrorHandler(handler.logger),
		AuthenticateMiddleware(handler.logger, handler.AccessTokenValidator),
		IdempotencyKeyMiddleware(),
		MaxBodySizeMiddleware(10*1024), // 10kb
	)

	inference.POST("", handler.HandleUserRequests)

}
