package public

import (
	"context"
	"time"
)

type CreateJobRequest struct {
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}

type JobPayloadRequest struct {
	JobID  string
	UserID string
}

type JobPayloadResponse struct {
	JobID           string
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type JobService interface {
	Create(ctx context.Context, req CreateJobRequest) (string, error)
	PayloadByID(ctx context.Context, payloadReq JobPayloadRequest) (*JobPayloadResponse, error)
	MarkCompleted(ctx context.Context, jobID string, retryCount int) error
	MarkFailed(ctx context.Context, jobID string, retryCount int) error
	UpdateStatus(ctx context.Context, jobID, status string, retryCount int) error
	UpdateStatusIfStatus(ctx context.Context, jobID string, retryCount int, status, expectedStatus string) (bool, error)
	GetRetryPolicy(ctx context.Context, userID string) (RetryPolicy, error)
}

type JobSnapshot struct {
	JobID      string
	UserID     string
	Status     string
	RetryCount int
	CreatedAt  time.Time
}

type JobReader interface {
	GetByID(ctx context.Context, jobID string) (JobSnapshot, error)
	PendingOrphans(ctx context.Context, before time.Time, limit int) ([]JobSnapshot, error)
}
