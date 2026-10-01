package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type StreamMetrics struct {
	delivery *promclient.HistogramVec
	endToEnd *promclient.HistogramVec
}

var _ public.StreamMetricsRecorder = (*StreamMetrics)(nil)

func NewStreamMetrics(builder *MetricBuilder) (*StreamMetrics, error) {
	delivery := builder.newHistogramVecWithBuckets("stream_delivery_seconds", "Duration of writing a stream event to the HTTP response.", shortDurationBuckets)
	endToEnd := builder.newHistogramVecWithBuckets("job_end_to_end_seconds", "Time from job creation until terminal event delivery.", longDurationBuckets)
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register stream metrics: %w", err)
	}
	return &StreamMetrics{delivery: delivery, endToEnd: endToEnd}, nil
}

func (m *StreamMetrics) ObserveStreamDelivery(d time.Duration) {
	observe(m.delivery.WithLabelValues(), d)
}
func (m *StreamMetrics) ObserveJobEndToEnd(d time.Duration) {
	observe(m.endToEnd.WithLabelValues(), d)
}
