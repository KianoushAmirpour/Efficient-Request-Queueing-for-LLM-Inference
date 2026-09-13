package public

import (
	"context"
)

type NextJobClaim struct {
	JobID  string
	UserID string
}

type JobClaimer interface {
	NextJob(ctx context.Context, workerID int) (*NextJobClaim, error)
}
