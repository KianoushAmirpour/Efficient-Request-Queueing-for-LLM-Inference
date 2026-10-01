package adapters

import (
	"time"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	metricsPublicAPI "efficient-request-queueing-for-llm-inference/internal/observability/metrics/public"
)

type MetricsAdapter struct {
	recorder metricsPublicAPI.AdmissionMetricsRecorder
}

var _ domain.MetricsRecorder = MetricsAdapter{}

func NewMetricsAdapter(recorder metricsPublicAPI.AdmissionMetricsRecorder) MetricsAdapter {
	return MetricsAdapter{recorder: recorder}
}

func (a MetricsAdapter) RequestRejected(reason domain.RejectionReason) {
	if a.recorder == nil {
		return
	}
	switch reason {
	case domain.RejectionTokenLimit:
		a.recorder.ObserveAdmissionTokenLimitRejection()
	case domain.RejectionModelNotAllowed:
		a.recorder.ObserveAdmissionModelNotAllowedRejection()
	case domain.RejectionRateLimited:
		a.recorder.ObserveAdmissionRateLimitedRejection()
	default:
		a.recorder.ObserveAdmissionOtherRejection()
	}
}

func (a MetricsAdapter) RequestFinished(d time.Duration) {
	if a.recorder != nil {
		a.recorder.RequestFinished(d)
	}
}
