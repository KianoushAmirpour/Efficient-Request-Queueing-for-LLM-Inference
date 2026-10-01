package domain

import "time"

type RejectionReason string

const (
	RejectionTokenLimit      RejectionReason = "token_limit"
	RejectionModelNotAllowed RejectionReason = "model_not_allowed"
	RejectionRateLimited     RejectionReason = "rate_limited"
	RejectionOther           RejectionReason = "other"
)

type MetricsRecorder interface {
	RequestRejected(RejectionReason)
	RequestFinished(time.Duration)
}
