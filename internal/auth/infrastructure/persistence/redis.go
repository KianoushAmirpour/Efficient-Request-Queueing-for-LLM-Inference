package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
)

type RedisStateTokenManager struct {
	redisClient *redis.Client
}

func NewRedisStateTokenManager(redisClient *redis.Client) *RedisStateTokenManager {
	return &RedisStateTokenManager{redisClient: redisClient}
}

func (m *RedisStateTokenManager) Store(ctx context.Context, state, provider string, ttl time.Duration) error {
	key := fmt.Sprintf("oauth2:%s:%s", provider, state)
	setResult := m.redisClient.Set(ctx, key, "1", ttl)
	if err := setResult.Err(); err != nil {
		return fmt.Errorf("cache OAuth state param: %w", err)
	}
	return nil
}

func (m *RedisStateTokenManager) Exists(ctx context.Context, state, provider string) error {
	key := fmt.Sprintf("oauth2:%s:%s", provider, state)
	getResult := m.redisClient.Get(ctx, key)
	if err := getResult.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.ErrStateParameterNotFound
		}
		return fmt.Errorf("retrieve OAuth state param: %w", err)
	}
	return nil
}

func (m *RedisStateTokenManager) Delete(ctx context.Context, state, provider string) error {
	key := fmt.Sprintf("oauth2:%s:%s", provider, state)
	err := m.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("clear OAuth state param: %w", err)
	}
	return nil
}
