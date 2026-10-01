package domain

// MetricsRecorder is a recovery-owned outbound port.
type MetricsRecorder interface {
	JobRequeued()
	JobRetried()
	JobCompleted()
	JobFailed()
}
