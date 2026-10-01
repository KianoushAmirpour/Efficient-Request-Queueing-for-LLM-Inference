package domain

type MetricsRecorder interface {
	JobCompleted()
	JobFailed()
}
