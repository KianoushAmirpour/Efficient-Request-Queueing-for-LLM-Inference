package domain

import (
	"errors"
)

var (
	ErrRateLimited           = errors.New("request rate limited")
	ErrCostExceedsCapacity   = errors.New("request cost (input tokens + max output tokens) exceeded the maximum allowed tokens for the user's tier")
	ErrModelNotAllowed       = errors.New("model not allowed for the user's current tier")
	ErrInputTooLarge         = errors.New("input exceeds the maximum allowed token count for the user's tier")
	ErrContextLengthExceeded = errors.New("input tokens + max output tokens exceed the model's context window")
)
