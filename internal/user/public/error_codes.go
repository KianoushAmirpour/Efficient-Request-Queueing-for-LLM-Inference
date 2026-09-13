package public

// Error codes for the user module.
// These codes are part of the public API contract.
const (
	// #nosec G101 - error type constant, not a credential
	ErrCodeAdminTokenGenFailed      string = "ADMIN_JWT_GENERATION_FAILED"
	ErrCodeUserNotAdmin             string = "USER_NOT_ADMIN"
	ErrCodeAdminNotFound            string = "ADMIN_NOT_FOUND"
	ErrCodeUserNotFound             string = "USER_NOT_FOUND"
	ErrCodeInvalidateUserTierFailed string = "USER_TIER_INVALIDATION_FAILED"
	ErrCodeRegisterFailed           string = "USER_REGISTRATION_FAILED"
	ErrCodeTierFetchFailed          string = "USER_TIER_FETCH_FAILED"
	ErrCodeRoleFetchFailed          string = "USER_ROLE_FETCH_FAILED"
	ErrCodeAdminCreationFailed      string = "ADMIN_CREATION_FAILED"
)
