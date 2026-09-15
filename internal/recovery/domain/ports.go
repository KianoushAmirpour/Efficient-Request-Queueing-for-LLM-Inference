package domain

import (
	"context"
	"errors"
	"time"
)

const StatusCreated = "created"

var ErrJobNotFound = errors.New("job not found")

type Queue interface {
	ExpiredJobIDs(ctx context.Context, cutoffMs int64, limit int) ([]string, error)
	RemoveIfExpired(ctx context.Context, jobID string, cutoffMs int64) (bool, error)
	RequeueIfExpired(ctx context.Context, jobID, userID string, cutoffMs int64) (bool, error)
	CompletedJobIDs(ctx context.Context) ([]string, error)
	RemoveCompletedJob(ctx context.Context, jobID string) error
	EnqueueIfAbsent(ctx context.Context, jobID, userID string, capacity int) (bool, error)
}

type Job struct {
	JobID, UserID, Status string
	RetryCount            int
}

type OrphanJob struct {
	Job
	CreatedAt time.Time
}

type Jobs interface {
	ByID(ctx context.Context, jobID string) (Job, error)
	PendingOrphans(ctx context.Context, before time.Time, limit int) ([]OrphanJob, error)
}

type StatusWriter interface {
	MarkCompleted(ctx context.Context, jobID string, retryCount int) error
	MarkFailed(ctx context.Context, jobID string, retryCount int) error
	UpdateCreated(ctx context.Context, jobID string, retryCount int) (bool, error)
}

type RetryPolicy struct{ MaxAttempts int }

type Policies interface {
	GetRetryPolicy(ctx context.Context, userID string) (RetryPolicy, error)
}

type IdempotencyFailure interface {
	TransitionStatus(ctx context.Context, userID, jobID, status string) error
}

type Events interface {
	PublishEvent(ctx context.Context, jobID, eventType, data string) error
	Close(ctx context.Context, jobID string) error
}
