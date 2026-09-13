package domain

import (
	"time"
)

type UserPolicy struct {
	Tier string
}

type RetryPolicy struct {
	MaxAttempts int

	BaseDelay time.Duration

	MaxDelay time.Duration
}

type JobPayload struct {
	JobID string

	Model string

	Prompt string

	MaxOutputTokens int
}

type Job struct {
	JobID string

	UserID string

	Status Status

	CurrentAttempt uint8

	CreatedAt time.Time

	RetryAttempt RetryPolicy
}
