package sharederr

import (
	"errors"
	"fmt"
)

type AppError struct {
	ErrCode string
	ErrType string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.ErrCode
	}
	if e.ErrCode == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s: %s", e.ErrCode, e.Err.Error())
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(errcode, errtype string, err error) *AppError {
	return &AppError{
		ErrCode: errcode,
		ErrType: errtype,
		Err:     err,
	}
}

func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.ErrCode == t.ErrCode
	}
	return false
}

func Code(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.ErrCode
	}
	return ""
}

func IsCode(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.ErrCode == code
	}
	return false
}
