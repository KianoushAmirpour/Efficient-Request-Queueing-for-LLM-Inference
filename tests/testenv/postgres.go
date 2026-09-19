package testenv

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgres "efficient-request-queueing-for-llm-inference/infra/pg_db"
)

func NewPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := postgres.CreatePostgresPool(ctx, postgres.PostgresConfig{
		Database: postgres.DatabaseConfig{
			Host:    envOr("POSTGRES_HOST", "localhost"),
			Port:    5432,
			DB:      envOr("POSTGRES_DB", "postgres_db"),
			SSLMode: "disable",
		},
		User:     envOr("POSTGRES_USER", "postgres"),
		Password: envOr("POSTGRES_PASSWORD", "123456789"),
		ConnectionPool: postgres.ConnectionPoolConfig{
			MaxConns:        8,
			MinConns:        1,
			MaxIdleTime:     time.Minute,
			MaxConnLifetime: time.Minute,
			ConnTimeout:     2 * time.Second,
		},
	})
	if err != nil {
		t.Skipf("PostgreSQL integration dependency unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
