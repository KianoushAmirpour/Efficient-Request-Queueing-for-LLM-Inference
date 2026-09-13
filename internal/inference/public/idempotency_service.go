package public

import "context"

type IdempotencyService interface {
	TransitionStatus(ctx context.Context, userID, jobID, status string) error
}
