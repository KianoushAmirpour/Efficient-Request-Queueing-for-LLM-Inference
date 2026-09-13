package domain

import "time"

type IdempotencyAction string

const (
	ActionNew               IdempotencyAction = "NEW"
	ActionDuplicateInFlight IdempotencyAction = "DUPLICATE_IN_FLIGHT"
	ActionReplay            IdempotencyAction = "REPLAY"
	ActionUnknown           IdempotencyAction = "UNKNOWN_STATE"
	ActionFailed            IdempotencyAction = "FAILED"
)

type IdempotencyResult struct {
	Action    IdempotencyAction
	Status    string
	JobID     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Err       error
}
