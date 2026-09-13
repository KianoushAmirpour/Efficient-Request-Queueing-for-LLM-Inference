package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type DatabaseConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	DB           int           `yaml:"db"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
}

type ConnectionPoolConfig struct {
	MaxActiveConns  int           `yaml:"max_active_conns"`
	MinIdleConns    int           `yaml:"min_idle_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	PoolTimeout     time.Duration `yaml:"pool_timeout"`
}

type RedisConfig struct {
	Database       DatabaseConfig       `yaml:"database"`
	ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
	Password       string               `yaml:"-"`
}

func ConnectToRedis(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        cfg.Password,
		DB:              cfg.Database.DB,
		PoolSize:        cfg.ConnectionPool.MaxActiveConns,
		MinIdleConns:    cfg.ConnectionPool.MinIdleConns,
		MaxIdleConns:    cfg.ConnectionPool.MaxIdleConns,
		ConnMaxIdleTime: cfg.ConnectionPool.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnectionPool.ConnMaxLifetime,
		ReadTimeout:     cfg.Database.ReadTimeout,
		WriteTimeout:    cfg.Database.WriteTimeout,
		PoolTimeout:     cfg.ConnectionPool.PoolTimeout,
		DialTimeout:     cfg.Database.DialTimeout,
		PoolFIFO:        true,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis at %s: %w", addr, err)
	}

	return rdb, nil
}

func Shutdown(ctx context.Context, client *redis.Client, logger *slog.Logger) error {
	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		logger.ErrorContext(
			ctx,
			"failed to close Redis client",
			"error", err,
		)
		return err
	}

	return nil
}
