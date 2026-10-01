package adapters

import (
	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
)

type MetricsAdapter struct {
	recorder         metricsPublicAPI.JobRecoveryMetricsRecorder
	retryRecorder    metricsPublicAPI.JobRecoveryRetryMetricsRecorder
	terminalRecorder metricsPublicAPI.JobTerminalMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublicAPI.JobRecoveryMetricsRecorder, retryRecorder metricsPublicAPI.JobRecoveryRetryMetricsRecorder, terminalRecorder metricsPublicAPI.JobTerminalMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{recorder: recorder, retryRecorder: retryRecorder, terminalRecorder: terminalRecorder}
}

func (a MetricsAdapter) JobRequeued() {
	if a.recorder != nil {
		a.recorder.ObserveJobRecoveredRequeued()
	}
}
func (a MetricsAdapter) JobRetried() {
	if a.retryRecorder != nil {
		a.retryRecorder.ObserveJobRetriedByRecovery()
	}
}
func (a MetricsAdapter) JobCompleted() {
	if a.terminalRecorder != nil {
		a.terminalRecorder.ObserveJobSucceeded()
	}
}
func (a MetricsAdapter) JobFailed() {
	if a.recorder != nil {
		a.recorder.ObserveJobRecoveredFailed()
	}
	if a.terminalRecorder != nil {
		a.terminalRecorder.ObserveJobFailed()
	}
}
