package adapters

import (
	"time"

	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
	"efficient-request-queueing-for-llm-inference/internal/scheduler/domain"
)

type MetricsAdapter struct {
	recorder metricsPublicAPI.SchedulerMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublicAPI.SchedulerMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{recorder: recorder}
}

func (a MetricsAdapter) ObserveQueueWaitDuration(duration time.Duration) {
	if a.recorder != nil {
		a.recorder.ObserveJobQueueWait(duration)
	}
}
