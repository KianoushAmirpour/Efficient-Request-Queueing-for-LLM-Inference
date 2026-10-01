package adapters

import (
	"time"

	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type MetricsAdapter struct {
	recorder         metricsPublicAPI.WorkerMetricsRecorder
	terminalRecorder metricsPublicAPI.JobTerminalMetricsRecorder
	retryRecorder    metricsPublicAPI.JobRetryMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(
	recorder metricsPublicAPI.WorkerMetricsRecorder,
	retryRecorder metricsPublicAPI.JobRetryMetricsRecorder,
	terminalRecorder metricsPublicAPI.JobTerminalMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{
		recorder:         recorder,
		retryRecorder:    retryRecorder,
		terminalRecorder: terminalRecorder}
}

func (a MetricsAdapter) JobStarted() {
	if a.recorder != nil {
		a.recorder.JobClaimed()
	}
}
func (a MetricsAdapter) JobEnded() {
	if a.recorder != nil {
		a.recorder.JobAttemptEnded()
	}
}
func (a MetricsAdapter) JobCompleted() {
	if a.terminalRecorder != nil {
		a.terminalRecorder.ObserveJobSucceeded()
	}
}
func (a MetricsAdapter) JobFailed() {
	if a.terminalRecorder != nil {
		a.terminalRecorder.ObserveJobFailed()
	}
}
func (a MetricsAdapter) JobRetriedModelFailure() {
	if a.retryRecorder != nil {
		a.retryRecorder.ObserveJobRetriedModelFailure()
	}
}
func (a MetricsAdapter) JobRetriedLeaseExpired() {
	if a.retryRecorder != nil {
		a.retryRecorder.ObserveJobRetriedLeaseExpired()
	}
}
func (a MetricsAdapter) ObserveJobProcessingSuccess(d time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveJobProcessingSuccess(d)
	}
}
func (a MetricsAdapter) ObserveJobProcessingFailure(d time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveJobProcessingFailure(d)
	}
}
