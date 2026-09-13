package scheduler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	infrastructure "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure"
	schedulerConfig "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	"efficient-request-queueing-for-llm-inference/internal/scheduler/usecase"
)

type SchedulerDeps struct {
	RedisClient  *redis.Client
	LeaseTimeout time.Duration
}

type SchedulerConfig = schedulerConfig.SchedulerConfig

type Module struct {
	QueueService      public.QueueService
	JobClaimerService public.JobClaimer
	RecoveryService   public.RecoveryService
}

func NewSchedulerModule(
	deps SchedulerDeps,
	cfg SchedulerConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("scheduler dependencies: redis client must not be nil")
	}
	if deps.LeaseTimeout <= 0 {
		return nil, fmt.Errorf("scheduler dependencies: lease timeout must be greater than zero")
	}
	if logger == nil {
		return nil, fmt.Errorf("scheduler logger must not be nil")
	}
	if cfg.QueueCapacity <= 0 {
		return nil, fmt.Errorf("scheduler configuration: queue capacity must be greater than zero")
	}
	if cfg.IdempotencyKeyTTL < time.Second {
		return nil, fmt.Errorf("scheduler configuration: idempotency key TTL must be at least one second")
	}

	schedulerRepo := infrastructure.NewRedisSchedulerRepository(deps.RedisClient, deps.LeaseTimeout, cfg)
	queueUsecase := usecase.NewQueueService(schedulerRepo)
	jobClaimerUseCase := usecase.NewJobClaimerUseCase(schedulerRepo)

	return &Module{
		QueueService:      queueUsecase,
		JobClaimerService: jobClaimerUseCase,
		RecoveryService:   schedulerRepo,
	}, nil
}

func (m *Module) Name() string { return "Scheduler" }
