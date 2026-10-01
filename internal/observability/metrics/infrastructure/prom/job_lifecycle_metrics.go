package prom

import (
	"fmt"

	promclient "github.com/prometheus/client_golang/prometheus"

	"efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type JobLifecycleMetrics struct {
	terminal  *promclient.CounterVec
	retried   *promclient.CounterVec
	recovered *promclient.CounterVec
}

var (
	_ public.JobTerminalMetricsRecorder      = (*JobLifecycleMetrics)(nil)
	_ public.JobRetryMetricsRecorder         = (*JobLifecycleMetrics)(nil)
	_ public.JobRecoveryRetryMetricsRecorder = (*JobLifecycleMetrics)(nil)
)

func NewJobLifecycleMetrics(builder *MetricBuilder) (*JobLifecycleMetrics, error) {
	terminal := builder.newCounterVec("jobs_terminal_total", "Jobs reaching a terminal state.", "status")
	retried := builder.newCounterVec("jobs_retried_total", "Jobs scheduled for another attempt.", "reason")
	recovered := builder.newCounterVec("jobs_recovered_total", "Jobs handled by recovery.", "result")
	if err := builder.err(); err != nil {
		return nil, fmt.Errorf("register job lifecycle metrics: %w", err)
	}
	for _, status := range []string{jobStatusSucceeded, jobStatusFailed} {
		terminal.WithLabelValues(status)
	}
	for _, reason := range []string{retryModelFailure, retryLeaseExpired, retryRecovery} {
		retried.WithLabelValues(reason)
	}
	for _, result := range []string{recoveryRequeued, recoveryFailed} {
		recovered.WithLabelValues(result)
	}
	return &JobLifecycleMetrics{terminal: terminal, retried: retried, recovered: recovered}, nil
}

func (m *JobLifecycleMetrics) ObserveJobSucceeded() {
	m.terminal.WithLabelValues(jobStatusSucceeded).Inc()
}
func (m *JobLifecycleMetrics) ObserveJobFailed() {
	m.terminal.WithLabelValues(jobStatusFailed).Inc()
}
func (m *JobLifecycleMetrics) ObserveJobRetriedModelFailure() {
	m.retried.WithLabelValues(retryModelFailure).Inc()
}
func (m *JobLifecycleMetrics) ObserveJobRetriedLeaseExpired() {
	m.retried.WithLabelValues(retryLeaseExpired).Inc()
}
func (m *JobLifecycleMetrics) ObserveJobRetriedByRecovery() {
	m.retried.WithLabelValues(retryRecovery).Inc()
}
