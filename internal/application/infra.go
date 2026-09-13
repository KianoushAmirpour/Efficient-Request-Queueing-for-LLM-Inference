package application

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	postgres "efficient-request-queueing-for-llm-inference/infra/pg_db"
	redisinfra "efficient-request-queueing-for-llm-inference/infra/redis"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

func newInfrastructure(
	ctx context.Context,
	logger *slog.Logger,
	cfg AppConfig,
) (*pgxpool.Pool, *redis.Client, error) {
	pgPool, err := postgres.CreatePostgresPool(ctx, *cfg.PostgresCfg)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"postgres setup failed",
			"code", sharederr.ErrCodeInternal,
			"type", "POSTGRES_CONNECTION_FAILED",
			"error", err,
		)
		return nil, nil, err
	}

	redisClient, err := redisinfra.ConnectToRedis(ctx, *cfg.RedisCfg)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"redis setup failed",
			"code", sharederr.ErrCodeInternal,
			"type", "REDIS_CONNECTION_FAILED",
			"error", err,
		)
		return nil, nil, err
	}

	return pgPool, redisClient, nil
}
