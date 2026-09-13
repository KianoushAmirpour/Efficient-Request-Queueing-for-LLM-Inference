package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthService struct {
	pgPool  *pgxpool.Pool
	redis   *redis.Client
	timeout time.Duration
}

func NewHealthService(pgPool *pgxpool.Pool, redisClient *redis.Client, timeout time.Duration) *HealthService {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HealthService{
		pgPool:  pgPool,
		redis:   redisClient,
		timeout: timeout,
	}
}

func (s *HealthService) CheckPostgres(ctx context.Context) error {
	if s == nil || s.pgPool == nil {
		return fmt.Errorf("postgres health check failed: client is not configured")
	}

	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.pgPool.Ping(checkCtx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}

func (s *HealthService) CheckRedis(ctx context.Context) error {
	if s == nil || s.redis == nil {
		return fmt.Errorf("redis health check failed: client is not configured")
	}

	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.redis.Ping(checkCtx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}

func (s *HealthService) Check(ctx context.Context) error {
	postgresErr := make(chan error, 1)
	redisErr := make(chan error, 1)

	go func() { postgresErr <- s.CheckPostgres(ctx) }()
	go func() { redisErr <- s.CheckRedis(ctx) }()

	if err := <-postgresErr; err != nil {
		return err
	}
	return <-redisErr
}
