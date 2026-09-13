package public

const (
	// #nosec G101 - error type constant, not a credential
	ErrCodeInputToken = "INPUT_TOKEN_EXCEEDS_TIER_LIMIT"

	ErrCodeModelNotAllowed = "MODEL_NOT_ALLOWED_FOR_TIER"

	ErrCodeRateLimited = "RATE_LIMIT_EXCEEDED"

	ErrCodeCostExceeded = "REQUEST_COST_EXCEEDS_TIER_LIMIT"

	ErrCodePolicyFetchFailed = "POLICY_FETCH_FAILED"
)
