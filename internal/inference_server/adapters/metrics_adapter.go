package adapters

import (
	"time"

	"efficient-request-queueing-for-llm-inference/internal/inference_server/domain"
	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type MetricsAdapter struct {
	recorder metricsPublicAPI.LLMGenerationMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublicAPI.LLMGenerationMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{recorder: recorder}
}

func (a MetricsAdapter) ObserveDuration(duration time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveGenerationDuration(duration)
	}
}

func (a MetricsAdapter) ObserveTimeToFirstToken(duration time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveTimeToFirstToken(duration)
	}
}
