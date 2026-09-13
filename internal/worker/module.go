package worker

import (
	"context"
	"fmt"
	"log/slog"

	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
	"efficient-request-queueing-for-llm-inference/internal/worker/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/worker/public"
	"efficient-request-queueing-for-llm-inference/internal/worker/usecase"
)

type WorkerDeps struct {
	JobClaimer         domain.JobClaimer
	JobService         domain.JobRepository
	InferenceService   domain.InferenceEngine
	StreamPublisher    domain.StreamPublisher
	QueueService       domain.Queue
	IdempotencyUpdater domain.IdempotencyStatusUpdater
}

type WorkerConfig = config.WorkerConfig

type Module struct {
	WorkerPool public.WorkerPool
}

func NewWorkerModule(
	deps WorkerDeps,
	cfg WorkerConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.JobClaimer == nil {
		return nil, fmt.Errorf("worker dependencies: job claimer must not be nil")
	}
	if deps.JobService == nil {
		return nil, fmt.Errorf("worker dependencies: job service must not be nil")
	}
	if deps.InferenceService == nil {
		return nil, fmt.Errorf("worker dependencies: inference service must not be nil")
	}
	if deps.StreamPublisher == nil {
		return nil, fmt.Errorf("worker dependencies: stream publisher must not be nil")
	}
	if deps.QueueService == nil {
		return nil, fmt.Errorf("worker dependencies: queue service must not be nil")
	}
	if deps.IdempotencyUpdater == nil {
		return nil, fmt.Errorf("worker dependencies: idempotency updater must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("worker logger must not be nil")
	}

	workerPool, err := usecase.NewWorkerPool(
		&cfg,
		deps.JobClaimer,
		deps.JobService,
		deps.InferenceService,
		deps.StreamPublisher,
		deps.QueueService,
		deps.IdempotencyUpdater,
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &Module{
		WorkerPool: workerPool,
	}, nil
}

func (m *Module) Name() string { return "Worker" }

func (m *Module) Start(ctx context.Context) error {
	return m.WorkerPool.Start(ctx)
}

func (m *Module) Stop(ctx context.Context) error {
	return m.WorkerPool.Stop(ctx)
}
