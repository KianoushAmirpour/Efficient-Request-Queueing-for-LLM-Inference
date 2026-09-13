package healthchecker

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/health_checker/handler"
	"efficient-request-queueing-for-llm-inference/internal/health_checker/service"
)

func RegisterHealthRoutes(rg *gin.RouterGroup, healthHandler *handler.HealthHandler) {
	if rg == nil || healthHandler == nil {
		return
	}

	health := rg.Group("/health")
	health.GET("/live", healthHandler.Liveness)
	health.GET("/ready", healthHandler.Readiness)
}

type Module struct {
	healthHandler *handler.HealthHandler
}

type HealthCheckerDeps struct {
	PGPool      *pgxpool.Pool
	RedisClient *redis.Client
}

type HealthCheckerConfig struct {
	Timeout time.Duration
}

func NewHealthCheckerModule(deps HealthCheckerDeps, cfg HealthCheckerConfig, logger *slog.Logger) (*Module, error) {
	if deps.PGPool == nil {
		return nil, fmt.Errorf("health checker dependencies: postgres pool must not be nil")
	}
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("health checker dependencies: redis client must not be nil")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("health checker config: timeout must be greater than zero")
	}
	if logger == nil {
		return nil, fmt.Errorf("health checker logger must not be nil")
	}

	return &Module{
		healthHandler: handler.NewHealthHandler(service.NewHealthService(deps.PGPool, deps.RedisClient, cfg.Timeout)),
	}, nil
}

func (m *Module) Name() string { return "Health Checker" }

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) error {
	RegisterHealthRoutes(rg, m.healthHandler)
	return nil
}
