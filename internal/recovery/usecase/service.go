package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
	recoveryErr "efficient-request-queueing-for-llm-inference/internal/recovery/errors"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeRecovery = "RECOVERY_FAILED"

type RecoveryService struct {
	queue         domain.Queue
	jobs          domain.Jobs
	statuses      domain.StatusWriter
	policies      domain.Policies
	idempotency   domain.IdempotencyFailure
	events        domain.Events
	interval      time.Duration
	batch         int
	orphanGrace   time.Duration
	queueCapacity int
	logger        *slog.Logger
	mu            sync.Mutex
	stop          context.CancelFunc
}

func NewRecoveryService(
	queue domain.Queue,
	jobs domain.Jobs,
	statuses domain.StatusWriter,
	policies domain.Policies,
	idempotency domain.IdempotencyFailure,
	events domain.Events,
	interval time.Duration,
	batch int,
	orphanGrace time.Duration,
	queueCapacity int,
	logger *slog.Logger) *RecoveryService {
	return &RecoveryService{
		queue:         queue,
		jobs:          jobs,
		statuses:      statuses,
		policies:      policies,
		idempotency:   idempotency,
		events:        events,
		interval:      interval,
		batch:         batch,
		orphanGrace:   orphanGrace,
		queueCapacity: queueCapacity,
		logger:        logger}
}

func (s *RecoveryService) ReconcileCompleted(ctx context.Context) error {
	ids, err := s.queue.CompletedJobIDs(ctx)
	if err != nil {
		return sharederr.EnsureAppError(err, recoveryErr.ErrCodeReconcileFailed, ErrTypeRecovery)
	}
	for _, id := range ids {
		job, err := s.jobs.ByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrJobNotFound) {
				_ = s.queue.RemoveCompletedJob(ctx, id)
				s.logger.WarnContext(ctx, "failed to remove completed job in recovery phase", "job.id", id, "error", err)
			}
			continue
		}
		if job.Status == domain.StatusCreated {
			if err := s.statuses.MarkCompleted(ctx, id, job.RetryCount); err != nil {
				s.logger.WarnContext(ctx, "failed to mark the job as completed in recovery phase", "job.id", id, "error", err)
				continue
			}
		}
		if err := s.idempotency.TransitionStatus(ctx, job.UserID, id, "completed"); err != nil {
			s.logger.WarnContext(ctx, "failed to update idempotency status in recovery phase", "job.id", id, "error", err)
			continue
		}
		if err := s.events.Close(ctx, id); err != nil {
			continue
		}
		if err := s.queue.RemoveCompletedJob(ctx, id); err != nil {
			return sharederr.EnsureAppError(err, recoveryErr.ErrCodeReconcileFailed, ErrTypeRecovery)
		}
	}
	return nil
}

func (s *RecoveryService) SweepOrphans(ctx context.Context) error {
	if s.orphanGrace <= 0 {
		s.orphanGrace = 5 * s.interval
	}
	if s.queueCapacity <= 0 {
		s.queueCapacity = 10
	}
	jobs, err := s.jobs.PendingOrphans(ctx, time.Now().Add(-s.orphanGrace), s.batch)
	if err != nil {
		return sharederr.EnsureAppError(err, recoveryErr.ErrCodeOrphanSweepFailed, ErrTypeRecovery)
	}
	for _, job := range jobs {
		queued, err := s.queue.EnqueueIfAbsent(ctx, job.JobID, job.UserID, s.queueCapacity)
		if err != nil {
			s.logger.WarnContext(ctx, "orphan enqueue failed", "job.id", job.JobID, "error", err)
			continue
		}
		if queued {
			s.logger.InfoContext(ctx, "orphan job re-enqueued", "job.id", job.JobID)
		}
	}
	return nil
}

func (s *RecoveryService) Sweep(ctx context.Context) error {
	now := time.Now().UnixMilli()
	ids, err := s.queue.ExpiredJobIDs(ctx, now, s.batch)
	if err != nil {
		return sharederr.NewAppError(recoveryErr.ErrCodeQueueRecoveryFailed, ErrTypeRecovery, err)
	}
	for _, id := range ids {
		if err := s.recoverOne(ctx, id, now); err != nil {
			s.logger.WarnContext(ctx, "recovery failed", "job.id", id, "error", err)
		}
	}
	return nil
}

func (s *RecoveryService) recoverOne(ctx context.Context, id string, cutoff int64) error {
	job, err := s.jobs.ByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			_, removeErr := s.queue.RemoveIfExpired(ctx, id, cutoff)
			return sharederr.EnsureAppError(removeErr, recoveryErr.ErrCodeQueueRecoveryFailed, ErrTypeRecovery)
		}
		return sharederr.NewAppError(recoveryErr.ErrCodeJobLookupFailed, ErrTypeRecovery, err)
	}

	if job.Status != domain.StatusCreated {
		_, err = s.queue.RemoveIfExpired(ctx, id, cutoff)
		return sharederr.EnsureAppError(err, recoveryErr.ErrCodeQueueRecoveryFailed, ErrTypeRecovery)
	}

	policy, err := s.policies.GetRetryPolicy(ctx, job.UserID)
	if err != nil {
		return sharederr.NewAppError(recoveryErr.ErrCodeRetryPolicyLookupFailed, ErrTypeRecovery, err)
	}

	next := job.RetryCount
	if next >= policy.MaxAttempts {
		claimed, err := s.queue.RemoveIfExpired(ctx, id, cutoff)
		if err != nil || !claimed {
			return sharederr.EnsureAppError(err, recoveryErr.ErrCodeQueueRecoveryFailed, ErrTypeRecovery)
		}
		if err = s.statuses.MarkFailed(ctx, id, next); err != nil {
			return sharederr.NewAppError(recoveryErr.ErrCodeJobStatusUpdateFailed, ErrTypeRecovery, err)
		}
		if err = s.idempotency.TransitionStatus(ctx, job.UserID, id, "failed"); err != nil {
			return sharederr.NewAppError(recoveryErr.ErrCodeIdempotencyUpdateFailed, ErrTypeRecovery, err)
		}
		return sharederr.EnsureAppError(s.events.PublishEvent(ctx, id, "failed", "recovery retry limit exceeded"), recoveryErr.ErrCodeEventPublishFailed, ErrTypeRecovery)
	}

	claimed, err := s.queue.RequeueIfExpired(ctx, id, job.UserID, cutoff)
	if err != nil || !claimed {
		return sharederr.EnsureAppError(err, recoveryErr.ErrCodeQueueRecoveryFailed, ErrTypeRecovery)
	}

	return sharederr.EnsureAppError(s.statuses.UpdateCreated(ctx, id, next), recoveryErr.ErrCodeJobStatusUpdateFailed, ErrTypeRecovery)
}

func (s *RecoveryService) Start(ctx context.Context) error {
	if s.interval <= 0 {
		s.interval = 30 * time.Second
	}
	if s.batch <= 0 {
		s.batch = 100
	}
	s.mu.Lock()
	if s.stop != nil {
		s.mu.Unlock()
		return nil
	}
	recoveryCtx, recoveryCancel := context.WithCancel(ctx)
	s.stop = recoveryCancel
	s.mu.Unlock()
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-recoveryCtx.Done():
				return
			case <-ticker.C:
				if err := s.Sweep(recoveryCtx); err != nil {
					s.logger.WarnContext(recoveryCtx, "recovery sweep failed", "error", err)
				}
				if err := s.ReconcileCompleted(recoveryCtx); err != nil {
					s.logger.WarnContext(recoveryCtx, "completed-job reconciliation failed", "error", err)
				}
				if err := s.SweepOrphans(recoveryCtx); err != nil {
					s.logger.WarnContext(recoveryCtx, "orphan sweep failed", "error", err)
				}
			}
		}
	}()
	return nil
}

func (s *RecoveryService) Stop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		s.stop()
		s.stop = nil
	}
	return nil
}
