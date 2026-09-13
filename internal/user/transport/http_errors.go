package transport

import (
	"net/http"

	authPublic "efficient-request-queueing-for-llm-inference/internal/auth/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	userPublic "efficient-request-queueing-for-llm-inference/internal/user/public"
)

const (
	MsgUserNotFound     = "The user account you are trying to update could not be found. Please verify the user ID and try again."
	MsgUserNotAdmin     = "Your account does not have administrator privileges. Contact an administrator if you need access."
	MsgUserUnauthorized = "Authentication is required to access user management endpoints. Your session may have expired or the provided token is invalid. Please sign in again."
)

var UserCodeRegistry = sharederr.CodeRegistry{
	userPublic.ErrCodeUserNotFound:        sharederr.ErrorMeta{StatusCode: http.StatusNotFound, Message: MsgUserNotFound},
	userPublic.ErrCodeUserNotAdmin:        sharederr.ErrorMeta{StatusCode: http.StatusNotFound, Message: MsgUserNotAdmin},
	userPublic.ErrCodeAdminTokenGenFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	userPublic.ErrCodeAdminNotFound:       sharederr.ErrorMeta{StatusCode: http.StatusForbidden, Message: MsgUserNotAdmin},
	userPublic.ErrCodeAdminCreationFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},

	authPublic.ErrCodeJWTTokenGenerationFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	authPublic.ErrCodeJWTTokenValidationFailed: sharederr.ErrorMeta{StatusCode: http.StatusUnauthorized, Message: MsgUserUnauthorized},

	sharederr.ErrCodeDBQueryFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	sharederr.ErrCodeUnauthorized:  sharederr.ErrorMeta{StatusCode: http.StatusUnauthorized, Message: MsgUserUnauthorized},
	sharederr.ErrCodeForbidden:     sharederr.ErrorMeta{StatusCode: http.StatusForbidden, Message: MsgUserNotAdmin},
}
