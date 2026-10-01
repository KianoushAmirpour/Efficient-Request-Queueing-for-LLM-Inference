package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type HTTPMetrics struct {
	requests *promclient.CounterVec
	duration *promclient.HistogramVec
	inFlight *promclient.GaugeVec
}

var _ public.HTTPMetricsRecorder = (*HTTPMetrics)(nil)

func NewHTTPMetrics(builder *MetricBuilder) (*HTTPMetrics, error) {
	m := &HTTPMetrics{
		requests: builder.newCounterVec("http_requests_total", "Total HTTP requests received", "method", "route", "status"),
		duration: builder.newHistogramVecWithBuckets("http_request_duration_seconds", "HTTP request duration.", longDurationBuckets, "method", "route"),
		inFlight: builder.newGaugeVec("http_requests_in_flight", "HTTP requests currently being handled.", "method", "route"),
	}
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register HTTP metrics: %w", err)
	}
	return m, nil
}

func (m *HTTPMetrics) ObserveHTTPRequest(method, route string, status int) {
	m.requests.WithLabelValues(normalizeHTTPMethod(method), normalizeHTTPRoute(route), normalizeHTTPStatus(status)).Inc()
}

func (m *HTTPMetrics) ObserveHTTPRequestDuration(method, route string, d time.Duration) {
	observe(m.duration.WithLabelValues(normalizeHTTPMethod(method), normalizeHTTPRoute(route)), d)
}

func (m *HTTPMetrics) IncHTTPInFlight(method, route string) {
	m.inFlight.WithLabelValues(normalizeHTTPMethod(method), normalizeHTTPRoute(route)).Inc()
}

func (m *HTTPMetrics) DecHTTPInFlight(method, route string) {
	m.inFlight.WithLabelValues(normalizeHTTPMethod(method), normalizeHTTPRoute(route)).Dec()
}
