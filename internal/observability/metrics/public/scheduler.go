package public

import "time"

// SchedulerMetricsRecorder records queue wait.
type SchedulerMetricsRecorder interface {
	ObserveJobQueueWait(time.Duration)
}
