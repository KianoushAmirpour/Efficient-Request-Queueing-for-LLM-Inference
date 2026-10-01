package domain

import (
	"context"
	"time"
)

// MetricsRecorder records stream-event delivery and job-creation-to-terminal latency.
type MetricsRecorder interface {
	ObserveStreamDelivery(time.Duration)
	ObserveJobEndToEnd(time.Duration)
}

// JobAcceptedAtReader provides the persisted acceptance time used by end-to-end metrics.
type JobAcceptedAtReader interface {
	GetJobAcceptedAt(ctx context.Context, jobID string) (time.Time, error)
}
