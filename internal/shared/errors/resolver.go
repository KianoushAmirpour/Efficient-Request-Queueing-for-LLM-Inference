package sharederr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

const msgFallbackInternal = "Your request could not be completed due to an unexpected server error. Please retry."

type ErrorMeta struct {
	StatusCode int
	Message    string
}

type CodeRegistry map[string]ErrorMeta

type ResolvedError struct {
	StatusCode int
	ErrorCode  string
	ErrorType  string
	Error      error
	Message    string
}

type ErrorResponse struct {
	Error struct {
		ErrorCode string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func fallbackInternal(err error, errType string) ResolvedError {
	return ResolvedError{
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  ErrCodeInternal,
		ErrorType:  errType,
		Error:      err,
		Message:    msgFallbackInternal,
	}
}

func ResolveError(err error, registry CodeRegistry, defaultErrorType string) ResolvedError {
	var appErr *AppError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var valErr validator.ValidationErrors
	var maxBytesErr *http.MaxBytesError

	if errors.As(err, &appErr) {
		if meta, ok := registry[appErr.ErrCode]; ok {
			return ResolvedError{
				StatusCode: meta.StatusCode,
				ErrorCode:  appErr.ErrCode,
				ErrorType:  appErr.ErrType,
				Error:      appErr.Err,
				Message:    meta.Message,
			}
		}
		return fallbackInternal(err, defaultErrorType)
	}

	if errors.As(err, &valErr) {
		return ResolvedError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrCodeValidation,
			ErrorType:  defaultErrorType,
			Error:      err,
			Message:    "Your request contains invalid or missing fields. Verify all required parameters are provided and correctly formatted.",
		}
	}
	if errors.As(err, &typeErr) {
		return ResolvedError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "INVALID_FIELD_TYPE",
			ErrorType:  defaultErrorType,
			Error:      err,
			Message:    "Your request contains fields with invalid types. Verify all field types match the expected format.",
		}
	}
	if errors.As(err, &syntaxErr) {
		return ResolvedError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "INVALID_JSON",
			ErrorType:  defaultErrorType,
			Error:      err,
			Message:    "Your request could not be processed because the JSON payload is malformed. Ensure your request body is valid JSON.",
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ResolvedError{
			StatusCode: http.StatusGatewayTimeout,
			ErrorCode:  "REQUEST_TIMEOUT",
			ErrorType:  defaultErrorType,
			Error:      err,
			Message:    "Your request took too long to process and has timed out. Please retry your request shortly.",
		}
	}

	if errors.As(err, &maxBytesErr) {
		return ResolvedError{
			StatusCode: http.StatusRequestEntityTooLarge,
			ErrorCode:  "REQUEST_TOO_LARGE",
			ErrorType:  defaultErrorType,
			Error:      err,
			Message:    "Your request payload is too large to process. Please reduce the size of your request and try again.",
		}
	}

	return fallbackInternal(err, defaultErrorType)
}

func EnsureAppError(err error, defaultErrorCode, defaultErrorType string) error {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return err
	}
	return NewAppError(defaultErrorCode, defaultErrorType, err)
}
