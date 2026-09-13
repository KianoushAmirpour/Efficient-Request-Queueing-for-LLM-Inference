package public

const (
	ErrCodeInvalidRequest       = "INFERENCE_INVALID_REQUEST"
	ErrCodeAuthenticationFailed = "INFERENCE_AUTHENTICATION_FAILED"
	ErrCodeModelNotFound        = "INFERENCE_MODEL_NOT_FOUND"
	ErrCodeContentFiltered      = "INFERENCE_CONTENT_FILTERED"
	ErrCodeMaxTokensExceeded    = "INFERENCE_MAX_TOKENS_EXCEEDED"

	ErrCodeServerOverloaded  = "INFERENCE_SERVER_OVERLOADED"
	ErrCodeRateLimitExceeded = "INFERENCE_RATE_LIMIT_EXCEEDED"

	ErrCodeStreamInterrupted = "INFERENCE_STREAM_INTERRUPTED"

	ErrCodeUnknown              = "INFERENCE_UNKNOWN_ERROR"
	ErrCodeIncompleteGeneration = "INFERENCE_INCOMPLETE_GENERATION"

	ErrCodeGenerationFailed = "INFERENCE_GENERATION_FAILED"
)
