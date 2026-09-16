package public

const (
	ErrCodeIdempotencyConflict    string = "INFERENCE_IDEMPOTENCY_CONFLICT"
	ErrCodeCheckIdempotencyFailed string = "IDEMPOTENCY_CHECK_FAILED"
	ErrCodeTryCoalescingFailed    string = "TRY_COALESCING_FAILED"
	ErrCodeCoalescingRejected     string = "COALESCING_REJECTED"
	ErrCodeRequestValidation      string = "REQUEST_VALIDATION_FAILED"
	ErrCodeJobCreationFailed      string = "JOB_CREATION_FAILED"
	ErrCodeIdempotencyKeyMissing  string = "IDEMPOTENCY_KEY_MISSING"
	ErrCodeIdempotencyUnknown     string = "IDEMPOTENCY_UNKNOWN_STATUS"
	ErrCodeAdmissionFailed        string = "ADMISSION_FAILED"
	ErrCodeEnqueueFailed          string = "INFERENCE_ENQUEUE_FAILED"
	ErrCodeReleaseCoalescing      string = "RELEASE_COALESCING_FAILED"
	ErrCodeReleaseIdempotencyKey  string = "RELEASE_IDEMPOTENCY_FAILED"
	ErrCodeIdempotencyJobFailed   string = "IDEMPOTENCY_JOB_FAILED"
	ErrCodeSubmitDeadlineExceeded string = "SUBMIT_DEADLINE_EXCEEDED"
)
