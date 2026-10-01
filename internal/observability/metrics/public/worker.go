package public

import "time"

// WorkerMetricsRecorder records in-flight jobs and per-attempt processing time.
type WorkerMetricsRecorder interface {
	JobClaimed()
	JobAttemptEnded()
	ObserveJobProcessingSuccess(time.Duration)
	ObserveJobProcessingFailure(time.Duration)
}
