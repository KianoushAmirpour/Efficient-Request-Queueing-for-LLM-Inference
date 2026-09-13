package domain

import (
	"context"
)

type JobQueue interface {
	PushLeft(ctx context.Context, entry QueueEntry) (*EnqueueResult, error)
	ClaimNextJob(ctx context.Context, workerID int) (*FairDequeueResult, error)
	ReleaseProcessingJob(ctx context.Context, jobID string) error
	ExtendProcessingLease(ctx context.Context, jobID string) error
	MarkJobCompleted(ctx context.Context, jobID string) error
}

type RecoveryQueue interface {
	ExpiredJobIDs(ctx context.Context, cutoffMs int64, limit int) ([]string, error)
	RemoveIfExpired(ctx context.Context, jobID string, cutoffMs int64) (bool, error)
	RequeueIfExpired(ctx context.Context, jobID, userID string, cutoffMs int64) (bool, error)
	CompletedJobIDs(ctx context.Context) ([]string, error)
	RemoveCompletedJob(ctx context.Context, jobID string) error
	EnqueueIfAbsent(ctx context.Context, jobID, userID string, capacity int) (bool, error)
}
