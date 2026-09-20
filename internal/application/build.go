package application

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	infraredis "efficient-request-queueing-for-llm-inference/infra/redis"
	logging "efficient-request-queueing-for-llm-inference/internal/observability/logger"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

func BuildApplication(rootctx context.Context) (*Application, error) {

	dbctx, dbctxCancel := context.WithTimeout(rootctx, 10*time.Second)
	defer dbctxCancel()

	log := logging.New(logging.Config{
		Level:     slog.LevelInfo,
		AddSource: false,
	})

	slog.SetDefault(log)

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.ErrorContext(
			rootctx,
			"loading env files failed",
			"code", sharederr.ErrCodeInternal,
			"type", "LOAD_CONFIG_FAILED",
			"error", err,
		)
		return nil, err
	}

	loader, err := newLoaderFromEnv()
	if err != nil {
		return nil, err
	}

	appCfg, err := loadConfigs(loader)
	if err != nil {
		return nil, err
	}

	pgPool, redisClient, err := newInfrastructure(dbctx, log, appCfg)
	if err != nil {
		return nil, err
	}

	moduleRegistry, err := composeModules(pgPool, redisClient, appCfg, log)
	if err != nil {
		pgPool.Close()
		if err := infraredis.Shutdown(dbctx, redisClient, log); err != nil {
			log.ErrorContext(
				dbctx,
				"failed to shut down redis",
				"error.code", sharederr.ErrCodeInternal,
				"error.type", "REDIS_SHUTDOWN_FAILED",
				"error", err,
			)
		}
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	return NewApplication(
		moduleRegistry,
		engine,
		log,
		pgPool,
		redisClient,
		*appCfg.ServerCfg,
	), nil
}
