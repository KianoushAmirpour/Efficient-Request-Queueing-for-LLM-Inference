package transport

import (
	"net/http"

	authPublic "efficient-request-queueing-for-llm-inference/internal/auth/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const (
	MsgSignInFetchFailed        = "We were unable to retrieve your GitHub account information. Please retry the sign-in process. If the issue continues, try again later."
	MsgSignInVerificationFailed = "Your sign-in session could not be verified. For security, please restart the sign-in process."
)

var AuthErrCodeRegistry = sharederr.CodeRegistry{
	authPublic.ErrCodeStateNotGenerated: sharederr.ErrorMeta{
		StatusCode: http.StatusInternalServerError,
		Message:    sharederr.MsgServerError},
	authPublic.ErrCodeFetchUserInfoFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusBadGateway,
		Message:    MsgSignInFetchFailed},
	authPublic.ErrCodeJWTTokenGenerationFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusInternalServerError,
		Message:    sharederr.MsgServerError},
	authPublic.ErrCodeUnauthorizedCallBack: sharederr.ErrorMeta{
		StatusCode: http.StatusUnauthorized,
		Message:    MsgSignInVerificationFailed},
	authPublic.ErrCodeTokenExchangeFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusBadGateway,
		Message:    MsgSignInFetchFailed},
	authPublic.ErrCodeJWTTokenValidationFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusUnauthorized,
		Message:    MsgSignInVerificationFailed},
	sharederr.ErrCodeCacheWriteFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusInternalServerError,
		Message:    sharederr.MsgServerError},
	sharederr.ErrCodeDBQueryFailed: sharederr.ErrorMeta{
		StatusCode: http.StatusInternalServerError,
		Message:    sharederr.MsgServerError},
}
