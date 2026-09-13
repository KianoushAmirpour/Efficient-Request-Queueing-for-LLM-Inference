package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
)

type TierRateLimit struct {
	Capacity int
	FillRate int
	TTL      time.Duration
}

type RateLimitConfig struct {
	Tiers map[string]TierRateLimit
}

type RedisRateLimiter struct {
	client *redis.Client
	cfg    RateLimitConfig
}

func NewRedisRateLimiter(client *redis.Client, cfg RateLimitConfig) RedisRateLimiter {
	return RedisRateLimiter{
		client: client,
		cfg:    cfg,
	}
}

func (r RedisRateLimiter) key(userID string) string {
	return fmt.Sprintf("llm:rate-limit:user:%s", userID)
}

var allowScript = redis.NewScript(`
-- KEYS[1]: rate limit key
-- ARGV[1]: capacity
-- ARGV[2]: fill rate
-- ARGV[3]: current timestamp in nanoseconds
-- ARGV[4]: TTL in seconds
-- ARGV[5]: cost

local key      = KEYS[1]

local capacity = tonumber(ARGV[1])
local fillRate = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local ttl      = tonumber(ARGV[4])
local cost     = tonumber(ARGV[5])

-- A request that can never fit the bucket: reject deterministically.
if cost > capacity then
  return -1
end

local data   = redis.call("HMGET", key, "tokens", "last")
local tokens = data[1]
local last   = data[2]

if not tokens then
  tokens = capacity
  last   = now
else
  tokens = tonumber(tokens)
  last   = tonumber(last)
  local elapsed = math.max(0, now - last) / 1e9
  tokens = math.min(capacity, tokens + elapsed * fillRate)
end

local allowed = 0
if tokens >= cost then
  tokens  = tokens - cost
  allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "last", now)
redis.call("EXPIRE", key, ttl)

return allowed
`)

func (r RedisRateLimiter) Allow(
	ctx context.Context,
	userID string,
	tier string,
	inputTokens, maxOutputTokens int,
) error {
	tierCfg, ok := r.cfg.Tiers[tier]
	if !ok {
		tierCfg, ok = r.cfg.Tiers["free"]
		if !ok {
			return fmt.Errorf("no rate limit configuration for tier %q and no fallback", tier)
		}
	}

	now := time.Now().UnixNano()
	key := r.key(userID)
	cost := inputTokens + maxOutputTokens

	res, err := allowScript.Run(
		ctx, r.client,
		[]string{key},
		tierCfg.Capacity, tierCfg.FillRate, now,
		int(tierCfg.TTL.Seconds()),
		cost,
	).Int()
	if err != nil {
		return fmt.Errorf("rate limiter redis: %w", err)
	}

	switch res {
	case -1:
		return domain.ErrCostExceedsCapacity
	case 1:
		return nil
	default:
		return domain.ErrRateLimited
	}
}
