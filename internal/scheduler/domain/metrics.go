package domain

import "time"

// MetricsRecorder is a scheduler-owned outbound port.
type MetricsRecorder interface {
	ObserveQueueWaitDuration(time.Duration)
}
