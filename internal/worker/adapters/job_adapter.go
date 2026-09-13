package adapters

import (
	"context"

	jobPublicAPI "efficient-request-queueing-for-llm-inference/internal/job/public"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type JobRepositoryAdapter struct {
	jobService jobPublicAPI.JobService
}

var _ domain.JobRepository = JobRepositoryAdapter{}

func NewJobRepositoryAdapter(jobService jobPublicAPI.JobService) JobRepositoryAdapter {
	return JobRepositoryAdapter{
		jobService: jobService,
	}
}

func (a JobRepositoryAdapter) Payload(ctx context.Context, claim *domain.JobClaimResult) (*domain.JobPayload, error) {
	payloadReq := jobPublicAPI.JobPayloadRequest{
		JobID:  claim.JobID,
		UserID: claim.UserID,
	}

	payloadResp, err := a.jobService.PayloadByID(ctx, payloadReq)
	if err != nil {
		return nil, err
	}
	return &domain.JobPayload{
		JobID:           payloadResp.JobID,
		UserID:          payloadResp.UserID,
		Prompt:          payloadResp.Prompt,
		Model:           payloadResp.Model,
		MaxOutputTokens: payloadResp.MaxOutputTokens,
	}, nil
}

func (a JobRepositoryAdapter) GetRetryPolicy(ctx context.Context, userID string) (domain.InferenceRetryPolicy, error) {
	policy, err := a.jobService.GetRetryPolicy(ctx, userID)
	if err != nil {
		return domain.InferenceRetryPolicy{}, err
	}

	return domain.InferenceRetryPolicy{
		MaxAttempts: policy.MaxAttempts,
		BaseDelay:   policy.BaseDelay,
		MaxDelay:    policy.MaxDelay,
	}, nil
}

func (a JobRepositoryAdapter) MarkCompleted(ctx context.Context, jobID string, retryCount int) error {
	return a.jobService.MarkCompleted(ctx, jobID, retryCount)
}

func (a JobRepositoryAdapter) MarkFailed(ctx context.Context, jobID string, retryCount int) error {
	return a.jobService.MarkFailed(ctx, jobID, retryCount)
}
