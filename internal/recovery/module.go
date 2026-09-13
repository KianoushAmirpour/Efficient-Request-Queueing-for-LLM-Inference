package recovery

import (
	"context"
	"fmt"
	"log/slog"

	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
	recoveryConfig "efficient-request-queueing-for-llm-inference/internal/recovery/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/recovery/usecase"
)

type Module struct{ service *usecase.RecoveryService }

type RecoveryDeps struct {
	Queue       domain.Queue
	Jobs        domain.Jobs
	Statuses    domain.StatusWriter
	Policies    domain.Policies
	Idempotency domain.IdempotencyFailure
	Events      domain.Events
}

func NewRecoveryModule(deps RecoveryDeps, cfg recoveryConfig.RecoveryConfig, logger *slog.Logger) (*Module, error) {
	if deps.Queue == nil {
		return nil, fmt.Errorf("recovery dependencies: queue must not be nil")
	}
	if deps.Jobs == nil {
		return nil, fmt.Errorf("recovery dependencies: jobs must not be nil")
	}
	if deps.Statuses == nil {
		return nil, fmt.Errorf("recovery dependencies: statuses must not be nil")
	}
	if deps.Policies == nil {
		return nil, fmt.Errorf("recovery dependencies: policies must not be nil")
	}
	if deps.Idempotency == nil {
		return nil, fmt.Errorf("recovery dependencies: idempotency must not be nil")
	}
	if deps.Events == nil {
		return nil, fmt.Errorf("recovery dependencies: events must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("recovery logger must not be nil")
	}
	if cfg.SweepInterval <= 0 {
		return nil, fmt.Errorf("recovery config: sweep_interval must be greater than zero")
	}
	if cfg.BatchSize <= 0 {
		return nil, fmt.Errorf("recovery config: batch_size must be greater than zero")
	}
	if cfg.OrphanGrace <= 0 {
		return nil, fmt.Errorf("recovery config: orphan_grace must be greater than zero")
	}
	if cfg.QueueCapacity <= 0 {
		return nil, fmt.Errorf("recovery config: queue_capacity must be greater than zero")
	}

	return &Module{service: usecase.NewRecoveryService(
		deps.Queue,
		deps.Jobs,
		deps.Statuses,
		deps.Policies,
		deps.Idempotency,
		deps.Events,
		cfg.SweepInterval,
		cfg.BatchSize,
		cfg.OrphanGrace,
		cfg.QueueCapacity,
		logger,
	)}, nil
}
func (m *Module) Name() string                    { return "Recovery" }
func (m *Module) Start(ctx context.Context) error { return m.service.Start(ctx) }
func (m *Module) Stop(ctx context.Context) error  { return m.service.Stop(ctx) }
