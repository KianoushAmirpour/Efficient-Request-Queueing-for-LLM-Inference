package prom

import (
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type WorkerMetrics struct {
	inFlight   promclient.Gauge
	processing *promclient.HistogramVec
}

var _ public.WorkerMetricsRecorder = (*WorkerMetrics)(nil)

func NewWorkerMetrics(builder *MetricBuilder) (*WorkerMetrics, error) {
	inFlight := builder.newGauge("worker_inflight_jobs", "Jobs currently being processed by workers.")
	processing := builder.newHistogramVecWithBuckets("job_processing_seconds", "Duration of an individual worker attempt.", longDurationBuckets, "status")
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register worker metrics: %w", err)
	}
	for _, status := range []string{processingSuccess, processingFailed} {
		processing.WithLabelValues(status)
	}
	return &WorkerMetrics{inFlight: inFlight, processing: processing}, nil
}

func (m *WorkerMetrics) JobClaimed()      { m.inFlight.Inc() }
func (m *WorkerMetrics) JobAttemptEnded() { m.inFlight.Dec() }
func (m *WorkerMetrics) ObserveJobProcessingSuccess(d time.Duration) {
	observe(m.processing.WithLabelValues(processingSuccess), d)
}
func (m *WorkerMetrics) ObserveJobProcessingFailure(d time.Duration) {
	observe(m.processing.WithLabelValues(processingFailed), d)
}
