package public

const (
	ErrCodeStateNotGenerated   = "OAUTH_STATE_PARAM_NOT_GENERATED"
	ErrCodeFetchUserInfoFailed = "OAUTH_USER_INFO_FAILED"
	// #nosec G101 - error type constant, not a credential
	ErrCodeJWTTokenGenerationFailed = "AUTH_JWT_TOKEN_GENERATION_FAILED"
	ErrCodeUnauthorizedCallBack     = "OAUTH_STATE_PARAM_NOT_FOUND"
	// #nosec G101 - error type constant, not a credential
	ErrCodeTokenExchangeFailed = "OAUTH_TOKEN_EXCHANGE_FAILED"
	// #nosec G101 - error type constant, not a credential
	ErrCodeJWTTokenValidationFailed = "AUTH_JWT_TOKEN_VALIDATION_FAILED"
)
