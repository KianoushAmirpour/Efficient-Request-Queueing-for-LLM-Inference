package domain

import "errors"

var (
	ErrInvalidRequest       = errors.New("invalid request")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrModelNotFound        = errors.New("model not found")
	ErrContentFiltered      = errors.New("content filtered")
	ErrMaxTokensExceeded    = errors.New("max tokens exceeded")

	ErrServerOverloaded  = errors.New("server overloaded")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	ErrStreamInterrupted = errors.New("stream interrupted")

	ErrUnknown              = errors.New("unknown error")
	ErrIncompleteGeneration = errors.New("incomplete generation")
)
