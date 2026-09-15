package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	inferenceConfig "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	inferencePublic "efficient-request-queueing-for-llm-inference/internal/inference/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeInference = "INFERENCE_FAILED"

type SubmitInferenceUseCase struct {
	admissionService   ports.RequestAdmitter
	idempotencyService ports.IdempotencyStore
	hashService        ports.Hasher
	coalescingService  ports.Coalescer
	jobService         ports.JobService
	queueService       ports.Enqueuer
	config             inferenceConfig.InferenceConfig
	logger             *slog.Logger
}

func NewInferenceUseCase(
	admissionSrv ports.RequestAdmitter,
	idempotencySrv ports.IdempotencyStore,
	hashSrv ports.Hasher,
	coalescingSrv ports.Coalescer,
	jobSrv ports.JobService,
	queueSrv ports.Enqueuer,
	cfg inferenceConfig.InferenceConfig,
	logger *slog.Logger,
) *SubmitInferenceUseCase {
	return &SubmitInferenceUseCase{
		admissionService:   admissionSrv,
		idempotencyService: idempotencySrv,
		hashService:        hashSrv,
		coalescingService:  coalescingSrv,
		jobService:         jobSrv,
		queueService:       queueSrv,
		config:             cfg,
		logger:             logger,
	}
}

func (i *SubmitInferenceUseCase) Submit(ctx context.Context, inferenceInput *domain.InferenceInput, userID, idempotencyHeader string) (jobID string, err error) {

	validatedTask, valErr := domain.ValidateInferenceInput(inferenceInput)
	if valErr != nil {
		return "", sharederr.EnsureAppError(valErr, inferencePublic.ErrCodeRequestValidation, ErrTypeInference)
	}
	validatedTask.UserID = userID

	idempotentResult, idempotencyErr := i.idempotencyService.ClaimOrGet(ctx, userID, idempotencyHeader, i.config.IdempotencyTTL)
	if idempotencyErr != nil {
		return "", sharederr.EnsureAppError(idempotencyErr, inferencePublic.ErrCodeCheckIdempotencyFailed, ErrTypeInference)
	}

	switch idempotentResult.Action {
	case domain.ActionNew:
	case domain.ActionDuplicateInFlight:
		return "", sharederr.EnsureAppError(idempotentResult.Err, inferencePublic.ErrCodeIdempotencyConflict, ErrTypeInference)
	case domain.ActionReplay:
		return idempotentResult.JobID, nil
	case domain.ActionUnknown:
		return "", sharederr.EnsureAppError(idempotentResult.Err, inferencePublic.ErrCodeIdempotencyUnknown, ErrTypeInference)
	case domain.ActionFailed:
		return "", sharederr.EnsureAppError(idempotentResult.Err, inferencePublic.ErrCodeIdempotencyJobFailed, ErrTypeInference)
	default:
		return "", sharederr.EnsureAppError(fmt.Errorf("unrecognized idempotency action: %s", idempotentResult.Action), inferencePublic.ErrCodeCheckIdempotencyFailed, ErrTypeInference)
	}

	hashRequest := i.hashService.HashRequest(validatedTask.GenerateKey())
	coalescingDecision, coalescingErr := i.coalescingService.TryCoalesce(ctx, userID, hashRequest, i.config.CoalescingTTL)
	if coalescingErr != nil {
		return "", sharederr.EnsureAppError(coalescingErr, inferencePublic.ErrCodeTryCoalescingFailed, ErrTypeInference)
	}

	if coalescingDecision.Action == domain.REJECTED || coalescingDecision.Action == domain.UNKNOWN {
		return "", sharederr.EnsureAppError(coalescingDecision.Error, inferencePublic.ErrCodeCoalescingRejected, ErrTypeInference)
	}

	admittedReq, admitErr := i.admissionService.Admit(ctx, validatedTask)
	if admitErr != nil {
		err := i.cleanupClaims(ctx, userID, idempotencyHeader, hashRequest)
		if err != nil {
			return "", err
		}
		return "", sharederr.EnsureAppError(admitErr, inferencePublic.ErrCodeAdmissionFailed, ErrTypeInference)
	}

	job, err := i.jobService.Create(ctx, admittedReq)
	if err != nil {
		err := i.cleanupClaims(ctx, userID, idempotencyHeader, hashRequest)
		if err != nil {
			return "", err
		}
		return "", sharederr.EnsureAppError(err, inferencePublic.ErrCodeJobCreationFailed, ErrTypeInference)
	}

	err = i.idempotencyService.SetJobID(ctx, userID, idempotencyHeader, job.JobID)
	if err != nil {
		if markErr := i.jobService.MarkFailed(ctx, job.JobID, job.CurrentAttempt); markErr != nil {
			i.logger.WarnContext(ctx, "failed to compensate job after idempotency binding failure", "job.id", job.JobID, "error", markErr)
		}
		err := i.cleanupClaims(ctx, userID, idempotencyHeader, hashRequest)
		if err != nil {
			return "", err
		}
		return "", sharederr.EnsureAppError(err, inferencePublic.ErrCodeCheckIdempotencyFailed, ErrTypeInference)
	}

	_, err = i.queueService.TryEnqueue(
		ctx,
		domain.InferenceJob{
			JobID:          job.JobID,
			UserID:         job.UserID,
			CurrentAttempt: job.CurrentAttempt,
		})
	if err != nil {
		if errors.Is(err, domain.ErrQueueFull) {
			if markErr := i.jobService.MarkFailed(ctx, job.JobID, job.CurrentAttempt); markErr != nil {
				i.logger.WarnContext(ctx, "failed to mark queue-full job failed", "job.id", job.JobID, "error", markErr)
			}
			_ = i.idempotencyService.TransitionStatus(ctx, job.JobID, "failed")
			remCoalErr := i.coalescingService.Delete(ctx, userID, hashRequest)
			if remCoalErr != nil {
				i.logger.WarnContext(ctx, "failed to delete coalescing key after queue-full rejection", "error", remCoalErr)
			}
		}
		return "", sharederr.EnsureAppError(err, inferencePublic.ErrCodeEnqueueFailed, ErrTypeInference)
	}

	return job.JobID, nil
}

func (i *SubmitInferenceUseCase) cleanupClaims(ctx context.Context, userID, idempotencyHeader string, hashRequest uint64) error {
	var err []error
	remCoalErr := i.coalescingService.Delete(ctx, userID, hashRequest)
	if remCoalErr != nil {
		i.logger.WarnContext(ctx, "failed to delete the coalescing key", "error", remCoalErr)
		err = append(err, sharederr.EnsureAppError(remCoalErr, inferencePublic.ErrCodeReleaseCoalescing, ErrTypeInference))
	}
	remIdempErr := i.idempotencyService.Delete(ctx, userID, idempotencyHeader)
	if remIdempErr != nil {
		i.logger.WarnContext(ctx, "failed to delete the idempotency key", "error", remIdempErr)
		err = append(err, sharederr.EnsureAppError(remIdempErr, inferencePublic.ErrCodeReleaseIdempotencyKey, ErrTypeInference))
	}
	return errors.Join(err...)
}
