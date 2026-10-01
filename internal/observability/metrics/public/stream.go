package public

import "time"

// StreamMetricsRecorder records event delivery and job-creation-to-terminal time.
type StreamMetricsRecorder interface {
	ObserveStreamDelivery(time.Duration)
	ObserveJobEndToEnd(time.Duration)
}
