package public

import "context"

type IdempotencyService interface {
	TransitionStatus(ctx context.Context, jobID, status string) error
	TransitionStatusIfInFlight(ctx context.Context, jobID, status string) (bool, error)
}
