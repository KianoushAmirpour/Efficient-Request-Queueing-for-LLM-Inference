package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

func MetricsMiddleware(recorder metricsPublicAPI.HTTPMetricsRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recorder == nil {
			c.Next()
			return
		}

		start := time.Now()
		method := c.Request.Method
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		recorder.IncHTTPInFlight(method, route)
		defer func() {
			recorder.DecHTTPInFlight(method, route)

			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			status := c.Writer.Status()
			duration := time.Since(start)

			recorder.ObserveHTTPRequest(method, route, status)
			recorder.ObserveHTTPRequestDuration(method, route, duration)
		}()

		c.Next()
	}
}
