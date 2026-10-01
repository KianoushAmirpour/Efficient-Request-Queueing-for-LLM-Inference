package public

import "time"

// AdmissionMetricsRecorder records admission decision latency and outcomes.
type AdmissionMetricsRecorder interface {
	RequestFinished(time.Duration)
	ObserveAdmissionTokenLimitRejection()
	ObserveAdmissionModelNotAllowedRejection()
	ObserveAdmissionRateLimitedRejection()
	ObserveAdmissionOtherRejection()
}
