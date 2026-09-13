package inference

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/coalescing"
	inferenceConfig "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/hash"
	"efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/idempotency"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	"efficient-request-queueing-for-llm-inference/internal/inference/public"
	transport "efficient-request-queueing-for-llm-inference/internal/inference/transport"
	"efficient-request-queueing-for-llm-inference/internal/inference/usecase"
)

type InferenceDeps struct {
	RedisClient            *redis.Client
	TokenValidationService ports.AccessTokenValidator
	AdmissionService       ports.RequestAdmitter
	JobService             ports.JobService
	QueueService           ports.Enqueuer
}

type InferenceConfig = inferenceConfig.InferenceConfig

type Module struct {
	InferenceHandler *transport.SubmitInferenceHandler
	IdompService     public.IdempotencyService
}

func NewInferenceModule(
	deps InferenceDeps,
	cfg InferenceConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("inference dependencies: redis client must not be nil")
	}
	if deps.TokenValidationService == nil {
		return nil, fmt.Errorf("inference dependencies: token validation service must not be nil")
	}
	if deps.AdmissionService == nil {
		return nil, fmt.Errorf("inference dependencies: admission service must not be nil")
	}
	if deps.JobService == nil {
		return nil, fmt.Errorf("inference dependencies: job service must not be nil")
	}
	if deps.QueueService == nil {
		return nil, fmt.Errorf("inference dependencies: queue service must not be nil")
	}
	if cfg.CoalescingTTL <= 0 {
		return nil, fmt.Errorf("inference configuration: coalescing TTL must be positive")
	}
	if cfg.IdempotencyTTL <= 0 {
		return nil, fmt.Errorf("inference configuration: idempotency TTL must be positive")
	}
	if cfg.CompletedIdempotencyTTL <= 0 {
		return nil, fmt.Errorf("inference configuration: completed idempotency TTL must be positive")
	}
	if logger == nil {
		return nil, fmt.Errorf("inference logger must not be nil")
	}

	idempotencyService := idempotency.NewRedisStore(deps.RedisClient, cfg)
	coalescingService := coalescing.NewCoalescingService(deps.RedisClient)
	hashingService := hash.NewHasher()
	inferenceUseCase := usecase.NewInferenceUseCase(
		deps.AdmissionService,
		idempotencyService,
		hashingService,
		coalescingService,
		deps.JobService,
		deps.QueueService,
		cfg,
		logger,
	)
	return &Module{
		InferenceHandler: transport.NewSubmitInferenceHandler(inferenceUseCase, deps.TokenValidationService, logger),
		IdompService:     idempotencyService,
	}, nil

}

func (m *Module) Name() string { return "Inference" }

func (m *Module) RegisterRoutes(api *gin.RouterGroup) error {

	transport.RegisterInferenceRoutes(api, m.InferenceHandler)

	return nil
}
