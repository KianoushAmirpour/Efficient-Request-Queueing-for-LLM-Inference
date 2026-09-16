package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/shared/retry"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
	"efficient-request-queueing-for-llm-inference/internal/worker/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/worker/public"
)

type workerPool struct {
	workerCount        int
	pollInterval       time.Duration
	leaseTimeout       time.Duration
	temperature        float32
	defaultMaxAttempts int
	defaultMaxDelay    time.Duration
	defaultBaseDelay   time.Duration
	maxRenewalFailures int

	jobClaimer         domain.JobClaimer
	jobRepo            domain.JobRepository
	inferenceEngine    domain.InferenceEngine
	streamPublisher    domain.StreamPublisher
	queueService       domain.Queue
	idempotencyUpdater domain.IdempotencyStatusUpdater
	logger             *slog.Logger

	wg     sync.WaitGroup
	cancel context.CancelFunc

	startMu sync.Mutex
	started bool
}

func NewWorkerPool(
	workerCfg *config.WorkerConfig,
	jobClaimer domain.JobClaimer,
	jobRepo domain.JobRepository,
	inferenceEngine domain.InferenceEngine,
	streamPublisher domain.StreamPublisher,
	queueService domain.Queue,
	idempotencyUpdater domain.IdempotencyStatusUpdater,
	logger *slog.Logger,
) (public.WorkerPool, error) {
	if workerCfg == nil {
		return nil, fmt.Errorf("worker pool: configuration must not be nil")
	}
	if workerCfg.WorkerCounts <= 0 {
		return nil, fmt.Errorf("worker pool: WorkerCounts must be > 0, got %d", workerCfg.WorkerCounts)
	}
	if workerCfg.PollInterval <= 0 {
		return nil, fmt.Errorf("worker pool: PollInterval must be > 0, got %s", workerCfg.PollInterval)
	}
	if workerCfg.LeaseTimeout <= 0 {
		return nil, fmt.Errorf("worker pool: LeaseTimeout must be > 0, got %s", workerCfg.LeaseTimeout)
	}
	if workerCfg.DefaultMaxAttempts == 0 {
		return nil, fmt.Errorf("worker pool: DefaultMaxAttempts must be > 0")
	}
	if workerCfg.DefaultBaseDelay <= 0 {
		return nil, fmt.Errorf("worker pool: DefaultBaseDelay must be > 0, got %s", workerCfg.DefaultBaseDelay)
	}
	if workerCfg.DefaultMaxDelay <= 0 {
		return nil, fmt.Errorf("worker pool: DefaultMaxDelay must be > 0, got %s", workerCfg.DefaultMaxDelay)
	}
	if workerCfg.DefaultMaxDelay < workerCfg.DefaultBaseDelay {
		return nil, fmt.Errorf("worker pool: DefaultMaxDelay must be >= DefaultBaseDelay")
	}
	if workerCfg.MaxRenewalFailures <= 0 {
		return nil, fmt.Errorf("worker pool: maxRenewalFailures must be > 0, got %d", workerCfg.MaxRenewalFailures)
	}

	return &workerPool{
		workerCount:        workerCfg.WorkerCounts,
		pollInterval:       workerCfg.PollInterval,
		leaseTimeout:       workerCfg.LeaseTimeout,
		temperature:        workerCfg.Temperature,
		defaultMaxAttempts: workerCfg.DefaultMaxAttempts,
		defaultMaxDelay:    workerCfg.DefaultMaxDelay,
		defaultBaseDelay:   workerCfg.DefaultBaseDelay,
		jobClaimer:         jobClaimer,
		jobRepo:            jobRepo,
		inferenceEngine:    inferenceEngine,
		streamPublisher:    streamPublisher,
		queueService:       queueService,
		idempotencyUpdater: idempotencyUpdater,
		logger:             logger}, nil
}

func (w *workerPool) Start(parent context.Context) error {
	w.startMu.Lock()
	defer w.startMu.Unlock()

	if w.started {
		return fmt.Errorf("worker pool: already started")
	}

	workerCtx, workerCancel := context.WithCancel(parent)

	w.cancel = workerCancel
	w.started = true

	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)

		go func(workerID int) {
			defer w.wg.Done()
			w.runWorker(workerCtx, workerID)
		}(i)
	}

	return nil
}

func (w *workerPool) Stop(ctx context.Context) error {
	w.startMu.Lock()

	if !w.started {
		w.startMu.Unlock()
		return nil
	}

	w.started = false

	cancel := w.cancel
	w.cancel = nil

	w.startMu.Unlock()

	cancel()

	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"worker pool: shutdown timed out: %w",
			ctx.Err(),
		)
	}
}

func (w *workerPool) wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true

	case <-ctx.Done():
		return false
	}
}

func (w *workerPool) runWorker(ctx context.Context, workerID int) {
	w.logger.InfoContext(ctx, "worker started", "worker.id", workerID)

	for {
		if ctx.Err() != nil {
			w.logger.InfoContext(ctx, "worker shutting down", "worker.id", workerID)
			return
		}

		err := w.processJob(ctx, workerID)

		switch {
		case err == nil:
			continue

		case errors.Is(err, domain.ErrNoJobAvailable):
			if !w.wait(ctx, w.pollInterval) {
				return
			}

		case errors.Is(err, context.Canceled):
			return

		default:
			w.logger.ErrorContext(
				ctx,
				"worker processing error",
				"worker.id", workerID,
				"error", err,
			)

			if !w.wait(ctx, w.pollInterval) {
				return
			}
		}
		continue
	}
}

func (w *workerPool) processJob(ctx context.Context, workerID int) error {

	claim, err := w.jobClaimer.ClaimNextJob(ctx, workerID)
	if err != nil {
		return err
	}

	if claim == nil {
		return domain.ErrNoJobAvailable
	}

	jobCtx, jobCancel := context.WithCancelCause(context.Background())
	defer jobCancel(nil)

	w.logger.InfoContext(jobCtx, "claimed job", "worker.id", workerID, "job.id", claim.JobID)

	heartbeatCtx, stopHeartbeat := context.WithCancelCause(jobCtx)
	heartbeatDone := make(chan struct{})
	go func() {

		defer close(heartbeatDone)

		interval := w.leaseTimeout / 3
		if interval <= 0 {
			interval = time.Second
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var consecutiveFailures int
		for {
			select {
			case <-ticker.C:
				err := w.queueService.ExtendLease(heartbeatCtx, claim.JobID)
				if err == nil {
					consecutiveFailures = 0
					continue
				}

				if errors.Is(err, domain.ErrLeaseExtendFailed) {
					w.logger.WarnContext(
						heartbeatCtx,
						"failed to renew processing lease",
						"worker.id", workerID,
						"job.id", claim.JobID,
						"error", err)
					jobCancel(fmt.Errorf("%w: lease no longer held", domain.ErrLeaseExtendFailed))
					return
				}

				consecutiveFailures++
				w.logger.WarnContext(jobCtx,
					"failed to renew processing lease; will retry",
					"worker.id", workerID, "job.id", claim.JobID,
					"consecutive_failures", consecutiveFailures,
					"max_consecutive_failures", w.maxRenewalFailures,
					"error", err)

				if consecutiveFailures >= w.maxRenewalFailures {
					w.logger.ErrorContext(jobCtx,
						"failed to renew processing lease; max consecutive failures reached; aborting job",
						"worker.id", workerID,
						"job.id", claim.JobID,
						"consecutive_failures", consecutiveFailures,
						"max_consecutive_failures", w.maxRenewalFailures,
						"error", err)

					jobCancel(fmt.Errorf("%w: max consecutive failures reached", domain.ErrLeaseExtendFailed))
					return
				}

			case <-heartbeatCtx.Done():
				return
			}
		}
	}()

	stopHeartbeatAndWait := func() {
		stopHeartbeat(nil)
		<-heartbeatDone
	}

	cleanupCtx := context.WithoutCancel(jobCtx)

	retryPolicy, err := w.jobRepo.GetRetryPolicy(jobCtx, claim.UserID)
	if err != nil {
		w.logger.WarnContext(
			jobCtx,
			"failed to get inference retry policy, using default",
			"worker.id", workerID,
			"job.id", claim.JobID,
			"user.id", claim.UserID,
			"error", err,
		)
		retryPolicy = domain.InferenceRetryPolicy{
			MaxAttempts: w.defaultMaxAttempts,
			BaseDelay:   w.defaultBaseDelay,
			MaxDelay:    w.defaultMaxDelay,
		}
	}

	payload, payloadErr := retry.DoWithRetry(
		jobCtx,
		retry.RetryPolicy{
			MaxAttempts: int(retryPolicy.MaxAttempts),
			BaseDelay:   retryPolicy.BaseDelay,
			MaxDelay:    retryPolicy.MaxDelay,
			ShouldRetry: sharederr.IsTransient,
			OnAttemptFailed: func(event retry.RetryEvent) {
				attrs := []any{
					"worker.id", workerID,
					"job.id", claim.JobID,
					"attempt", event.Attempt,
					"max_attempts", event.MaxAttempts,
					"error", event.Err,
				}

				if event.WillRetry {
					attrs = append(attrs, "next_delay", event.NextDelay)
					w.logger.WarnContext(jobCtx, "job payload fetch failed, will retry", attrs...)
				} else {
					w.logger.ErrorContext(jobCtx, "job payload fetch failed, no more retries", attrs...)
				}
			},
		},
		func(jobCtx context.Context) (*domain.JobPayload, error) {
			return w.jobRepo.Payload(jobCtx, claim)
		},
	)

	if payloadErr != nil {

		stopHeartbeatAndWait()

		if cause := context.Cause(jobCtx); errors.Is(cause, domain.ErrLeaseExtendFailed) {
			w.logger.ErrorContext(
				jobCtx,
				"aborted: processing lease lost; skipping cleanup to avoid clobbering the new owner",
				"worker.id", workerID,
				"job.id", claim.JobID)

			return fmt.Errorf("%w: %w", domain.ErrLeaseExtendFailed, payloadErr)
		}

		if markErr := w.jobRepo.MarkFailed(cleanupCtx, claim.JobID, int(retryPolicy.MaxAttempts)); markErr != nil {
			w.logger.ErrorContext(jobCtx, "failed to mark job as failed", "worker.id", workerID, "job.id", claim.JobID, "error", markErr)
		}

		if err := w.idempotencyUpdater.TransitionStatus(cleanupCtx, claim.UserID, claim.JobID, "failed"); err != nil {
			w.logger.ErrorContext(jobCtx, "failed to mark idempotency as failed", "worker.id", workerID, "job.id", claim.JobID, "error", err)
		}

		if err := w.queueService.Release(cleanupCtx, claim.JobID); err != nil {
			w.logger.ErrorContext(
				jobCtx,
				"failed to release job from processing set",
				"worker.id", workerID,
				"job.id", claim.JobID,
				"error", err,
			)
		}

		if w.streamPublisher != nil {
			message := `{"message":"We could not complete your request after several attempts. Please try again later."}`
			if publishErr := w.streamPublisher.PublishEvent(cleanupCtx, claim.JobID, "failed", message); publishErr != nil {
				w.logger.ErrorContext(jobCtx, "failed to publish terminal inference failure", "worker.id", workerID, "job.id", claim.JobID, "error", publishErr)
			}
		}

		return err
	}

	var inferenceAttempts int
	inferenceRetryPolicy := retry.RetryPolicy{
		MaxAttempts: int(retryPolicy.MaxAttempts),
		BaseDelay:   retryPolicy.BaseDelay,
		MaxDelay:    retryPolicy.MaxDelay,
		ShouldRetry: func(err error) bool {
			return errors.Is(err, domain.ErrTransientInference)
		},
		OnAttemptFailed: func(event retry.RetryEvent) {
			inferenceAttempts = event.Attempt
			attrs := []any{
				"worker.id", workerID,
				"job.id", claim.JobID,
				"attempt", event.Attempt,
				"max_attempts", event.MaxAttempts,
				"error", event.Err,
			}

			if event.WillRetry {
				if w.streamPublisher != nil {
					message := fmt.Sprintf(
						`{"attempt":%d,"max_attempts":%d,"message":"The inference service is temporarily unavailable. Retrying..."}`,
						event.Attempt,
						event.MaxAttempts,
					)
					if pubErr := w.streamPublisher.PublishEvent(jobCtx, claim.JobID, "retrying", message); pubErr != nil {
						w.logger.ErrorContext(jobCtx, "failed to publish inference retry status",
							"worker.id", workerID, "job.id", claim.JobID, "error", pubErr)
					}
				}
				attrs = append(attrs, "next_delay", event.NextDelay)
				w.logger.WarnContext(jobCtx, "inference attempt failed; retrying", attrs...)
			} else {
				w.logger.ErrorContext(jobCtx, "inference attempt failed; no retry remaining", attrs...)
			}
		},
	}

	genInput := domain.GenerationInput{
		Model:       payload.Model,
		Prompt:      payload.Prompt,
		MaxTokens:   payload.MaxOutputTokens,
		Temperature: w.temperature,
	}

	_, err = retry.DoWithRetry(
		jobCtx,
		inferenceRetryPolicy,
		func(ctx context.Context) (struct{}, error) {
			inferenceAttempts++
			genErr := w.inferenceEngine.GenerateStream(
				jobCtx,
				genInput,
				func(chunk domain.StreamChunk) error {
					if w.streamPublisher == nil {
						return nil
					}
					if pubErr := w.streamPublisher.Publish(jobCtx, claim.JobID, chunk); pubErr != nil {
						w.logger.ErrorContext(jobCtx, "failed to publish chunk",
							"worker.id", workerID, "job.id", claim.JobID, "error", pubErr)
						return nil
					}
					return nil
				})
			if genErr != nil {
				return struct{}{}, fmt.Errorf("%w: %w", domain.ErrInferenceFailed, genErr)
			}
			return struct{}{}, nil
		})

	retryCount := inferenceAttempts

	if err != nil {

		stopHeartbeatAndWait()

		if cause := context.Cause(jobCtx); errors.Is(cause, domain.ErrLeaseExtendFailed) {
			w.logger.ErrorContext(
				jobCtx,
				"aborted: processing lease lost; skipping cleanup to avoid clobbering the new owner",
				"worker.id", workerID,
				"job.id", claim.JobID)

			return fmt.Errorf("%w: %w", domain.ErrLeaseExtendFailed, err)
		}

		if markErr := w.jobRepo.MarkFailed(cleanupCtx, claim.JobID, retryCount); markErr != nil {
			w.logger.ErrorContext(
				jobCtx,
				"failed to mark job as failed after inference error",
				"worker.id", workerID,
				"job.id", claim.JobID,
				"error", markErr,
			)
		}

		if err := w.idempotencyUpdater.TransitionStatus(cleanupCtx, claim.UserID, claim.JobID, "failed"); err != nil {
			w.logger.ErrorContext(jobCtx, "failed to mark idempotency as failed", "worker.id", workerID, "job.id", claim.JobID, "error", err)
		}

		if err := w.queueService.Release(cleanupCtx, claim.JobID); err != nil {
			w.logger.ErrorContext(
				jobCtx,
				"failed to release job from processing set",
				"worker.id", workerID,
				"job.id", claim.JobID,
				"error", err,
			)
		}

		if w.streamPublisher != nil {
			message := `{"message":"We could not complete your request after several attempts. Please try again later."}`
			if publishErr := w.streamPublisher.PublishEvent(cleanupCtx, claim.JobID, "failed", message); publishErr != nil {
				w.logger.ErrorContext(jobCtx, "failed to publish terminal inference failure", "worker.id", workerID, "job.id", claim.JobID, "error", publishErr)
			}
		}

		return err
	}

	onSuccessCleanupCtx, onSuccessCleanupCancel := context.WithTimeout(
		context.WithoutCancel(jobCtx),
		10*time.Second,
	)
	defer onSuccessCleanupCancel()

	if w.queueService != nil {
		if markerErr := w.queueService.MarkCompleted(onSuccessCleanupCtx, claim.JobID); markerErr != nil {
			w.logger.ErrorContext(onSuccessCleanupCtx, "failed to mark execution completed", "job.id", claim.JobID, "error", markerErr)
		}
	}

	if err := w.jobRepo.MarkCompleted(onSuccessCleanupCtx, claim.JobID, retryCount); err != nil {
		w.logger.ErrorContext(
			onSuccessCleanupCtx,
			"failed to mark job as completed",
			"worker.id", workerID,
			"job.id", claim.JobID,
			"error", err,
		)

		stopHeartbeatAndWait()

		if w.streamPublisher != nil {
			_ = w.streamPublisher.Close(onSuccessCleanupCtx, claim.JobID)
		}
		_ = w.queueService.Release(onSuccessCleanupCtx, claim.JobID)
		return err
	}

	stopHeartbeatAndWait()

	if err := w.idempotencyUpdater.TransitionStatus(onSuccessCleanupCtx, claim.UserID, claim.JobID, "completed"); err != nil {
		w.logger.ErrorContext(onSuccessCleanupCtx, "failed to mark idempotency as completed", "worker.id", workerID, "job.id", claim.JobID, "error", err)
	}

	if w.streamPublisher != nil {
		if publishErr := w.streamPublisher.Close(onSuccessCleanupCtx, claim.JobID); publishErr != nil {
			w.logger.ErrorContext(onSuccessCleanupCtx, "failed to publish completed inference event", "worker.id", workerID, "job.id", claim.JobID, "error", publishErr)
		}
	}

	if err := w.queueService.Release(onSuccessCleanupCtx, claim.JobID); err != nil {
		w.logger.ErrorContext(
			onSuccessCleanupCtx,
			"failed to release job from processing set",
			"worker.id", workerID,
			"job.id", claim.JobID,
			"error", err,
		)
	}

	w.logger.InfoContext(onSuccessCleanupCtx, "completed job", "worker.id", workerID, "job.id", claim.JobID)
	return nil
}
