package prom

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type WorkloadMetrics struct{ accepted prometheus.Counter }

var _ public.WorkloadMetricsRecorder = (*WorkloadMetrics)(nil)

func NewWorkloadMetrics(builder *MetricBuilder) (*WorkloadMetrics, error) {
	accepted := builder.newCounterVec("jobs_accepted_total", "Jobs accepted for processing.")
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register workload metrics: %w", err)
	}
	return &WorkloadMetrics{accepted: accepted.WithLabelValues()}, nil
}

func (m *WorkloadMetrics) ObserveJobAccepted() { m.accepted.Inc() }
