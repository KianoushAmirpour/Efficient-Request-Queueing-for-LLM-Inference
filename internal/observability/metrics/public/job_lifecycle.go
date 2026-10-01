package public

// JobTerminalMetricsRecorder records terminal job outcomes.
type JobTerminalMetricsRecorder interface {
	ObserveJobSucceeded()
	ObserveJobFailed()
}

// JobRetryMetricsRecorder records retries initiated by worker outcomes.
type JobRetryMetricsRecorder interface {
	ObserveJobRetriedModelFailure()
	ObserveJobRetriedLeaseExpired()
}

// JobRecoveryRetryMetricsRecorder records retries initiated by recovery.
type JobRecoveryRetryMetricsRecorder interface {
	ObserveJobRetriedByRecovery()
}

// JobRecoveryMetricsRecorder records the result of recovering a job.
type JobRecoveryMetricsRecorder interface {
	ObserveJobRecoveredRequeued()
	ObserveJobRecoveredFailed()
}
