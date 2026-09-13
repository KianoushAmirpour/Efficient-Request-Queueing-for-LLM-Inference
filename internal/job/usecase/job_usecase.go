package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	jobDomain "efficient-request-queueing-for-llm-inference/internal/job/domain"
	"efficient-request-queueing-for-llm-inference/internal/job/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeJob string = "JOB_FAILED"

type JobUsecase struct {
	jobRepo     jobDomain.JobRepository
	userService jobDomain.UserPolicyReader
	tierPolicy  jobDomain.JobPolicyConfig
	uow         jobDomain.UnitOfWork
}

func NewJobUsecase(
	jobRepo jobDomain.JobRepository,
	tierPolicy jobDomain.JobPolicyConfig,
	userSvc jobDomain.UserPolicyReader,
	uow jobDomain.UnitOfWork,
) public.JobService {
	return &JobUsecase{
		jobRepo:     jobRepo,
		userService: userSvc,
		tierPolicy:  tierPolicy,
		uow:         uow,
	}
}

func (u *JobUsecase) Create(ctx context.Context, req public.CreateJobRequest) (string, error) {

	userPolicy, err := u.userService.GetUserPolicy(ctx, req.UserID)
	if err != nil {
		return "", sharederr.EnsureAppError(err, public.ErrCodePolicyFetchFailed, ErrTypeJob)
	}

	policy, ok := u.tierPolicy.Tiers[userPolicy.Tier]
	if !ok {
		return "", sharederr.NewAppError(
			public.ErrCodePolicyFetchFailed,
			ErrTypeJob,
			errors.New("no retry policy configured for tier: "+userPolicy.Tier),
		)
	}
	userRetryPolicy := jobDomain.RetryPolicy{
		MaxAttempts: policy.MaxAttempts,
		BaseDelay:   policy.BaseDelay,
		MaxDelay:    policy.MaxDelay,
	}

	jobID := uuid.New().String()

	job := &jobDomain.Job{
		JobID:          jobID,
		UserID:         req.UserID,
		Status:         jobDomain.StatusCreated,
		CurrentAttempt: 1,
		RetryAttempt:   userRetryPolicy,
	}

	jobReq := &jobDomain.JobPayload{
		JobID:           jobID,
		Model:           req.Model,
		Prompt:          req.Prompt,
		MaxOutputTokens: req.MaxOutputTokens,
	}

	result, err := u.uow.Execute(ctx, func(txCtx context.Context) (string, error) {
		if err := u.jobRepo.Save(txCtx, job); err != nil {
			return "", err
		}

		if err := u.jobRepo.SaveRequest(txCtx, jobReq); err != nil {
			return "", err
		}

		return job.JobID, nil
	})

	if err != nil {
		return "", sharederr.EnsureAppError(err, sharederr.ErrCodeDBTransactionFailed, ErrTypeJob)
	}

	return result, nil
}

func (u *JobUsecase) PayloadByID(ctx context.Context, payloadReq public.JobPayloadRequest) (*public.JobPayloadResponse, error) {

	jobPayload, err := u.jobRepo.PayloadByID(ctx, payloadReq.JobID)
	if err != nil {
		if errors.Is(err, jobDomain.ErrJobNotFound) {
			return nil, sharederr.EnsureAppError(err, public.ErrCodeJobNotFound, ErrTypeJob)
		}
		return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeJob)
	}

	return &public.JobPayloadResponse{
		JobID:           jobPayload.JobID,
		UserID:          payloadReq.UserID, // Use the UserID from the request instead of the payload to ensure consistency
		Prompt:          jobPayload.Prompt,
		Model:           jobPayload.Model,
		MaxOutputTokens: jobPayload.MaxOutputTokens,
	}, nil
}

func (u *JobUsecase) MarkCompleted(ctx context.Context, jobID string, retryCount int) error {
	job, err := u.jobRepo.JobByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, jobDomain.ErrJobNotFound) {
			return sharederr.EnsureAppError(err, public.ErrCodeJobNotFound, ErrTypeJob)
		}
		return sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeJob)
	}

	if err := job.Complete(); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeJobStatusUpdateFailed, ErrTypeJob)
	}

	if err := u.jobRepo.UpdateStatus(ctx, jobID, job.Status, retryCount); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeJobStatusUpdateFailed, ErrTypeJob)
	}

	return nil
}

func (u *JobUsecase) MarkFailed(ctx context.Context, jobID string, retryCount int) error {
	job, err := u.jobRepo.JobByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, jobDomain.ErrJobNotFound) {
			return sharederr.EnsureAppError(err, public.ErrCodeJobNotFound, ErrTypeJob)
		}
		return sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeJob)
	}

	if err := job.Fail(); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeJobStatusUpdateFailed, ErrTypeJob)
	}

	if err := u.jobRepo.UpdateStatus(ctx, jobID, job.Status, retryCount); err != nil {
		return sharederr.EnsureAppError(err, public.ErrCodeJobStatusUpdateFailed, ErrTypeJob)
	}

	return nil
}

func (u *JobUsecase) GetRetryPolicy(ctx context.Context, userID string) (public.RetryPolicy, error) {
	userPolicy, err := u.userService.GetUserPolicy(ctx, userID)
	if err != nil {
		return public.RetryPolicy{}, sharederr.EnsureAppError(err, public.ErrCodePolicyFetchFailed, ErrTypeJob)
	}

	tierPolicy, ok := u.tierPolicy.Tiers[userPolicy.Tier]
	if !ok {
		return public.RetryPolicy{}, sharederr.NewAppError(
			public.ErrCodePolicyFetchFailed,
			ErrTypeJob,
			errors.New("no retry policy configured for tier: "+userPolicy.Tier),
		)
	}

	return public.RetryPolicy{
		MaxAttempts: tierPolicy.MaxAttempts,
		BaseDelay:   tierPolicy.BaseDelay,
		MaxDelay:    tierPolicy.MaxDelay,
	}, nil
}

func (u *JobUsecase) GetByID(ctx context.Context, jobID string) (public.JobSnapshot, error) {
	job, err := u.jobRepo.JobByID(ctx, jobID)
	if err != nil {
		return public.JobSnapshot{}, err
	}
	return public.JobSnapshot{JobID: job.JobID, UserID: job.UserID, Status: string(job.Status), RetryCount: int(job.CurrentAttempt)}, nil
}

func (u *JobUsecase) PendingOrphans(ctx context.Context, before time.Time, limit int) ([]public.JobSnapshot, error) {
	jobs, err := u.jobRepo.PendingOrphans(ctx, before, limit)
	if err != nil {
		return nil, err
	}
	result := make([]public.JobSnapshot, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, public.JobSnapshot{JobID: job.JobID, UserID: job.UserID, Status: string(job.Status), RetryCount: int(job.CurrentAttempt), CreatedAt: job.CreatedAt})
	}
	return result, nil
}

func (u *JobUsecase) UpdateStatus(ctx context.Context, jobID, status string, retryCount int) error {
	return u.jobRepo.UpdateStatus(ctx, jobID, jobDomain.Status(status), retryCount)
}
