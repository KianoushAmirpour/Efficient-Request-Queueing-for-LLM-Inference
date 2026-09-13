package domain

import (
	"time"
)

type QueueEntry struct {
	JobID          string
	UserID         string
	CurrentAttempt int
	EnqueuedAt     time.Time
}

type EnqueueResult struct {
	BecameActive bool
}

type FairDequeueResult struct {
	UserID string
	JobID  string
}
