package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

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

	err := godotenv.Load()
	if err != nil {
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
