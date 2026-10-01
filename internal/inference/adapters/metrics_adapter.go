package adapters

import (
	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type MetricsAdapter struct {
	recorder metricsPublicAPI.WorkloadMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublicAPI.WorkloadMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{recorder: recorder}
}

func (a MetricsAdapter) JobAccepted() {
	if a.recorder != nil {
		a.recorder.ObserveJobAccepted()
	}
}
