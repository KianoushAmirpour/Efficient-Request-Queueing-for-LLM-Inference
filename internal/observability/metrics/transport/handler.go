package transport

import (
	"github.com/gin-gonic/gin"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricHandler struct {
	Registry promclient.Gatherer
}

func NewMetricHandler(registry promclient.Gatherer) *MetricHandler {
	return &MetricHandler{
		Registry: registry,
	}
}

func (h *MetricHandler) HandleMetrics(c *gin.Context) {
	handler := promhttp.HandlerFor(
		h.Registry,
		promhttp.HandlerOpts{},
	)

	handler.ServeHTTP(c.Writer, c.Request)
}
