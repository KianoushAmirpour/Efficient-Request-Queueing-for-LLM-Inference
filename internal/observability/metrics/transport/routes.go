package transport

import (
	"github.com/gin-gonic/gin"
)

func RegisterMetricsRoutes(rg *gin.RouterGroup, handler *MetricHandler) {
	rg.GET("/metrics", handler.HandleMetrics)
}
