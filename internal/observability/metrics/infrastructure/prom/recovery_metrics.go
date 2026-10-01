package prom

import (
	"github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type RecoveryMetrics struct {
	recovered *prometheus.CounterVec
}

var _ public.JobRecoveryMetricsRecorder = (*RecoveryMetrics)(nil)

func NewRecoveryMetrics(metrics *JobLifecycleMetrics) *RecoveryMetrics {
	return &RecoveryMetrics{recovered: metrics.recovered}
}

func (m *RecoveryMetrics) ObserveJobRecoveredRequeued() {
	m.recovered.WithLabelValues(recoveryRequeued).Inc()
}
func (m *RecoveryMetrics) ObserveJobRecoveredFailed() {
	m.recovered.WithLabelValues(recoveryFailed).Inc()
}
