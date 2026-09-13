package transport

import (
	"net/http"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	streamPublic "efficient-request-queueing-for-llm-inference/internal/stream/public"
)

const (
	MsgUnauthorized         = "Authentication is required to access stream endpoints. Your session may have expired or the provided token is invalid. Please sign in again and include a valid authentication token."
	MsgInternalError        = "Your stream request could not be completed due to an unexpected server error. Please retry. If the problem persists, include your request ID when contacting support."
	MsgStreamNotFound       = "The requested stream could not be found. Please verify the stream ID and try again."
	MsgInvalidStreamRequest = "Your stream request contains invalid parameters. Verify the stream ID format and subscription options, then try again."
	MsgServiceUnavailable   = "The streaming service is temporarily unavailable. Please retry in a few moments."
)

var StreamCodeRegistry = sharederr.CodeRegistry{
	streamPublic.ErrCodeStreamNotFound:    sharederr.ErrorMeta{StatusCode: http.StatusNotFound, Message: MsgStreamNotFound},
	streamPublic.ErrCodeSubscribeFailed:   sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: MsgInternalError},
	streamPublic.ErrCodePublishFailed:     sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: MsgInternalError},
	streamPublic.ErrCodeConnectionLost:    sharederr.ErrorMeta{StatusCode: http.StatusServiceUnavailable, Message: MsgServiceUnavailable},
	streamPublic.ErrCodeInvalidStreamData: sharederr.ErrorMeta{StatusCode: http.StatusBadRequest, Message: MsgInvalidStreamRequest},

	sharederr.ErrCodeInternal:     sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: MsgInternalError},
	sharederr.ErrCodeUnauthorized: sharederr.ErrorMeta{StatusCode: http.StatusUnauthorized, Message: MsgUnauthorized},
}
