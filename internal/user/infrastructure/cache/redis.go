package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const userTierCacheTTL = 15 * time.Minute

type Cacher struct {
	redisclient *redis.Client
}

func NewCacher(client *redis.Client) *Cacher {
	return &Cacher{
		redisclient: client,
	}
}

func buildUserTierCacheKey(userID string) string {
	return fmt.Sprintf("user:%s:tier", userID)
}

func (c *Cacher) Invalidate(ctx context.Context, userID string) error {
	key := buildUserTierCacheKey(userID)

	err := c.redisclient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("invalidate user tier cache: %w", err)
	}

	return nil
}

func (c *Cacher) Get(ctx context.Context, userID string) (string, error) {

	key := buildUserTierCacheKey(userID)

	userTier, err := c.redisclient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return userTier, nil

}

func (c *Cacher) Set(ctx context.Context, userID, userTier string) error {

	key := buildUserTierCacheKey(userID)

	writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.redisclient.Set(
		writeCtx,
		key,
		userTier,
		userTierCacheTTL,
	).Err()
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}
