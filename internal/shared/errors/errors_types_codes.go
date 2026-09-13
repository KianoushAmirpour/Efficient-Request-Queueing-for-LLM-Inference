package sharederr

import (
	"context"
	"errors"
	"net"
)

const (
	ErrCodeDBQueryFailed            string = "DB_QUERY_FAILED"
	ErrCodeDBTransactionFailed      string = "DB_TX_FAILED"
	ErrCodeCacheWriteFailed         string = "CACHE_WRITE_FAILED"
	ErrCodeCacheReadFailed          string = "CACHE_READ_FAILED"
	ErrCodeCacheDeleteFailed        string = "CACHE_DELETE_FAILED"
	ErrCodeLuaScriptExecutionFailed string = "LUA_SCRIPT_EXECUTION_FAILED"
	ErrCodeUnauthorized             string = "UNAUTHORIZED"
	ErrCodeInternal                 string = "INTERNAL_ERROR"
	ErrCodeValidation               string = "VALIDATION_ERROR"
	ErrCodeNotFound                 string = "NOT_FOUND"
	ErrCodeConflict                 string = "CONFLICT"
	ErrCodeForbidden                string = "FORBIDDEN"
	ErrCodeMarshalingStruct         string = "MARSHALING_FAILED"
)

const (
	MsgServerError string = "We couldn't complete your request due to an unexpected server error. Please retry. If the problem continues, include your request ID when contacting support."
)

func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
