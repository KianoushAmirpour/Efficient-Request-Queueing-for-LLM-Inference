package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	DB      string `yaml:"db"`
	SSLMode string `yaml:"sslmode"`
}

type ConnectionPoolConfig struct {
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxIdleTime     time.Duration `yaml:"max_idle_time"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	ConnTimeout     time.Duration `yaml:"conn_timeout"`
}

type PostgresConfig struct {
	Database       DatabaseConfig       `yaml:"database"`
	ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
	User           string               `yaml:"-"`
	Password       string               `yaml:"-"`
}

func CreatePostgresPool(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	if cfg.ConnectionPool.MaxIdleTime <= 0 {
		return nil, fmt.Errorf("postgres config: max_idle_time must be greater than zero")
	}
	if cfg.ConnectionPool.MaxConnLifetime <= 0 {
		return nil, fmt.Errorf("postgres config: max_conn_lifetime must be greater than zero")
	}
	if cfg.ConnectionPool.ConnTimeout <= 0 {
		return nil, fmt.Errorf("postgres config: conn_timeout must be greater than zero")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DB, cfg.Database.SSLMode)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	config.MinConns = cfg.ConnectionPool.MinConns
	config.MaxConns = cfg.ConnectionPool.MaxConns
	config.MaxConnLifetime = cfg.ConnectionPool.MaxConnLifetime
	config.MaxConnIdleTime = cfg.ConnectionPool.MaxIdleTime
	config.ConnConfig.ConnectTimeout = cfg.ConnectionPool.ConnTimeout

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
