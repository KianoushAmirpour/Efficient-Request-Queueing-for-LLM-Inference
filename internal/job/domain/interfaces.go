package domain

import (
	"context"
	"time"
)

type JobRepository interface {
	Save(ctx context.Context, job *Job) error

	SaveRequest(ctx context.Context, jobReq *JobPayload) error

	PayloadByID(ctx context.Context, jobID string) (*JobPayload, error)

	JobByID(ctx context.Context, jobID string) (*Job, error)
	PendingOrphans(ctx context.Context, before time.Time, limit int) ([]*Job, error)

	UpdateStatus(ctx context.Context, jobID string, status Status, retryCount int) error

	UpdateStatusIfStatus(ctx context.Context, jobID string, retryCount int, status, expectedStatus Status) (bool, error)
}

type UserPolicyReader interface {
	GetUserPolicy(ctx context.Context, userID string) (UserPolicy, error)
}
