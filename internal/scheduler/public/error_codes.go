package public

const (
	ErrCodeJobClaimerFailed        = "CLAIM_NEXT_JOB_FAILED"
	ErrCodeEnqueueFailed           = "ENQUEUE_FAILED"
	ErrCodeQueueFull               = "USER_QUEUE_IS_FULL"
	ErrCodeDuplicatedJob           = "DUPLICATE_JOB"
	ErrCodeReleaseProcessingFailed = "RELEASE_PROCESSING_JOB_FAILED"
	ErrCodeReleaseExtendingFailed  = "RELEASE_EXTENDING_FAILED"
)
