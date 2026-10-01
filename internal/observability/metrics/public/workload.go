package public

// WorkloadMetricsRecorder records successfully accepted jobs.
type WorkloadMetricsRecorder interface {
	ObserveJobAccepted()
}
