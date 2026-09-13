package transport

import (
	"net/http"

	admissionPublicAPI "efficient-request-queueing-for-llm-inference/internal/admission/public"
	inferencePublic "efficient-request-queueing-for-llm-inference/internal/inference/public"
	schedulerPublicAPI "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const (
	MsgValidationFailed = "Your inference request contains invalid or missing fields. Verify that the prompt is non-empty, the model name is correct, and all required parameters are provided."

	MsgUnauthorized = "Authentication is required to submit inference requests. Your session may have expired or the provided token is invalid. Please sign in again and include a valid authentication token."

	MsgIdempotencyConflict = "A request with the same idempotency key is already in progress. Wait for the current request to complete before retrying, or submit a new request with a different idempotency key."

	MsgRateLimited = "You have exceeded your allowed request rate. Please wait a moment before submitting another request. Consider upgrading your tier for higher rate limits."

	MsgModelNotAllowed = "The requested model is not available for your current access tier. Please choose a different model or upgrade your tier to access this model."
	// #nosec G101 - error type constant, not a credential
	MsgInputTokenExceeded = "Your prompt exceeds the maximum allowed input token count for your access tier. Shorten your prompt or upgrade your tier for higher limits."

	MsgCostExceeded = "The estimated cost of this request (input tokens plus maximum output tokens) exceeds your tier's capacity. Reduce the prompt size, lower the max output tokens, or upgrade your tier."

	MsgJobIdempotencyConflict = "This request is already being processed. Please wait for it to complete."

	MsgMaximumConcurrentRequests = "Service is unavailable for your tier now. Please wait."

	MsgCoalescingRejected = "A request with identical content is currently being processed. Please retry shortly."

	MsgIdempotencyKeyRequired = "The Idempotency-Key header is required for this endpoint. Include a unique key to safely retry requests without duplication."
)

var InferenceCodeRegistry = sharederr.CodeRegistry{
	inferencePublic.ErrCodeIdempotencyConflict:    sharederr.ErrorMeta{StatusCode: http.StatusConflict, Message: MsgIdempotencyConflict},
	inferencePublic.ErrCodeCheckIdempotencyFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	inferencePublic.ErrCodeTryCoalescingFailed:    sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	inferencePublic.ErrCodeCoalescingRejected:     sharederr.ErrorMeta{StatusCode: http.StatusConflict, Message: MsgCoalescingRejected},
	inferencePublic.ErrCodeRequestValidation:      sharederr.ErrorMeta{StatusCode: http.StatusBadRequest, Message: MsgValidationFailed},
	inferencePublic.ErrCodeJobCreationFailed:      sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	inferencePublic.ErrCodeIdempotencyKeyMissing:  sharederr.ErrorMeta{StatusCode: http.StatusBadRequest, Message: MsgIdempotencyKeyRequired},
	inferencePublic.ErrCodeIdempotencyUnknown:     sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	inferencePublic.ErrCodeAdmissionFailed:        sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},

	admissionPublicAPI.ErrCodeInputToken:        sharederr.ErrorMeta{StatusCode: http.StatusRequestEntityTooLarge, Message: MsgInputTokenExceeded},
	admissionPublicAPI.ErrCodeModelNotAllowed:   sharederr.ErrorMeta{StatusCode: http.StatusForbidden, Message: MsgModelNotAllowed},
	admissionPublicAPI.ErrCodeRateLimited:       sharederr.ErrorMeta{StatusCode: http.StatusTooManyRequests, Message: MsgRateLimited},
	admissionPublicAPI.ErrCodeCostExceeded:      sharederr.ErrorMeta{StatusCode: http.StatusPaymentRequired, Message: MsgCostExceeded},
	admissionPublicAPI.ErrCodePolicyFetchFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},

	schedulerPublicAPI.ErrCodeDuplicatedJob: sharederr.ErrorMeta{StatusCode: http.StatusConflict, Message: MsgJobIdempotencyConflict},
	schedulerPublicAPI.ErrCodeEnqueueFailed: sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	schedulerPublicAPI.ErrCodeQueueFull:     sharederr.ErrorMeta{StatusCode: http.StatusServiceUnavailable, Message: MsgMaximumConcurrentRequests},

	sharederr.ErrCodeInternal:     sharederr.ErrorMeta{StatusCode: http.StatusInternalServerError, Message: sharederr.MsgServerError},
	sharederr.ErrCodeUnauthorized: sharederr.ErrorMeta{StatusCode: http.StatusUnauthorized, Message: MsgUnauthorized},
}
