package domain

import "errors"

var (
	ErrJobNotFound = errors.New("job not found")

	ErrInvalidTransition = errors.New("invalid job transition")

	ErrAlreadyCompleted = errors.New("job already completed")

	ErrAlreadyFailed = errors.New("job already failed")
)
