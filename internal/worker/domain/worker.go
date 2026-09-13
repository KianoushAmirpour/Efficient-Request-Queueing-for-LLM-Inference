package domain

import (
	"context"
	"time"
)

type JobClaimResult struct {
	JobID  string
	UserID string
}

type JobPayload struct {
	JobID           string
	UserID          string
	Prompt          string
	Model           string
	MaxOutputTokens int
}

type GenerationInput struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float32
}

type StreamChunk struct {
	Text string
}

type StreamOutput struct {
	Text string
}

type InferenceRetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type JobClaimer interface {
	ClaimNextJob(ctx context.Context, workerID int) (*JobClaimResult, error)
}

type JobRepository interface {
	Payload(ctx context.Context, claim *JobClaimResult) (*JobPayload, error)
	GetRetryPolicy(ctx context.Context, userID string) (InferenceRetryPolicy, error)
	MarkCompleted(ctx context.Context, jobID string, retryCount int) error
	MarkFailed(ctx context.Context, jobID string, retryCount int) error
}

type InferenceEngine interface {
	GenerateStream(ctx context.Context, input GenerationInput, onChunk func(StreamChunk) error) error
}

type StreamPublisher interface {
	Publish(ctx context.Context, jobID string, chunk StreamChunk) error
	PublishEvent(ctx context.Context, jobID string, eventType string, data string) error
	Close(ctx context.Context, jobID string) error
}

type Queue interface {
	Release(ctx context.Context, jobID string) error
	ExtendLease(ctx context.Context, jobID string) error
	MarkCompleted(ctx context.Context, jobID string) error
}

type IdempotencyStatusUpdater interface {
	TransitionStatus(ctx context.Context, userID, jobID, status string) error
}
