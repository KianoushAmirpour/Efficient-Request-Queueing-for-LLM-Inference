package public

import "context"

type QueueEntry struct {
	JobID          string
	UserID         string
	CurrentAttempt int
}

type EnqueueResult struct {
	BecameActive bool
}

type QueueService interface {
	Enqueue(ctx context.Context, entry QueueEntry) (EnqueueResult, error)
	ReleaseProcessingJob(ctx context.Context, jobID string) error
	ExtendProcessingLease(ctx context.Context, jobID string) error
	MarkJobCompleted(ctx context.Context, jobID string) error
}

type RecoveryService interface {
	ExpiredJobIDs(ctx context.Context, cutoffMs int64, limit int) ([]string, error)
	RemoveIfExpired(ctx context.Context, jobID string, cutoffMs int64) (bool, error)
	RequeueIfExpired(ctx context.Context, jobID, userID string, cutoffMs int64) (bool, error)
	CompletedJobIDs(ctx context.Context) ([]string, error)
	RemoveCompletedJob(ctx context.Context, jobID string) error
	EnqueueIfAbsent(ctx context.Context, jobID, userID string, capacity int) (bool, error)
}
