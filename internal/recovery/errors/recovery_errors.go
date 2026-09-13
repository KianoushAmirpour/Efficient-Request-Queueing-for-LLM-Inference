package errors

const (
	ErrCodeQueueRecoveryFailed     = "RECOVERY_QUEUE_OPERATION_FAILED"
	ErrCodeJobLookupFailed         = "RECOVERY_JOB_LOOKUP_FAILED"
	ErrCodeRetryPolicyLookupFailed = "RECOVERY_RETRY_POLICY_LOOKUP_FAILED"
	ErrCodeJobStatusUpdateFailed   = "RECOVERY_JOB_STATUS_UPDATE_FAILED"
	ErrCodeIdempotencyUpdateFailed = "RECOVERY_IDEMPOTENCY_UPDATE_FAILED"
	ErrCodeEventPublishFailed      = "RECOVERY_EVENT_PUBLISH_FAILED"
	ErrCodeReconcileFailed         = "RECOVERY_RECONCILE_FAILED"
	ErrCodeOrphanSweepFailed       = "RECOVERY_ORPHAN_SWEEP_FAILED"
)
