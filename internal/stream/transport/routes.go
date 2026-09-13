package transport

import (
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterStreamRoutes(router *gin.RouterGroup, handler *StreamHTTPHandler) {
	stream := router.Group("/stream")
	stream.Use(
		TimeoutMiddleware(30*time.Minute),
		ErrorHandler(handler.logger),
		AuthenticateMiddleware(handler.AccessTokenValidator),
	)
	{
		stream.GET("/:jobID", handler.HandleStream)
	}
}
