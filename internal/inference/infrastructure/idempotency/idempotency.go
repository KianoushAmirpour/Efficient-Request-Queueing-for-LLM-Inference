package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	inferenceConfig "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
)

type RedisStore struct {
	redisClient *redis.Client
	config      inferenceConfig.InferenceConfig
}

func NewRedisStore(redisClient *redis.Client, cfg inferenceConfig.InferenceConfig) *RedisStore {
	return &RedisStore{
		redisClient: redisClient,
		config:      cfg,
	}
}

func buildIdempotencyKey(userID, idempotencyHeader string) string {
	return fmt.Sprintf("idempotent:%s:inference:%s", userID, idempotencyHeader)
}

var (
	idempotencyScript = redis.NewScript(`
-- KEYS[1]: idempotency key
-- ARGV[1]: TTL in seconds
-- ARGV[2]: current timestamp

local key = KEYS[1]
local ttl = tonumber(ARGV[1])
local now = ARGV[2]

-- Single atomic write of all fields, guarded by existence check
if redis.call("EXISTS", key) == 0 then
  redis.call("HSET", key, "status", "in_flight", "created_at", now)
  redis.call("EXPIRE", key, ttl)
  return {"NEW", "in_flight", now, ""}
end

-- Already exists: read current state
local status = redis.call("HGET", key, "status")
local created_at = redis.call("HGET", key, "created_at") or ""
local job_id = redis.call("HGET", key, "job_id") or ""

if status == "in_flight" then
  return {"DUPLICATE_IN_FLIGHT", status, created_at, job_id}
elseif status == "completed" then
  return {"REPLAY", status, created_at, job_id}
elseif status == "failed" then
  return {"UNKNOWN_STATE", status or "missing", created_at, job_id}
else
  return {"UNKNOWN_STATE", status or "missing", created_at, job_id}
end
`)

	setJobIDScript = redis.NewScript(`
-- KEYS[1] = The idempotency key
-- KEYS[2] = The reverse lookup key
-- ARGV[1] = The job_id string

local idem_key = KEYS[1]
local reverse_key = KEYS[2]
local job_id = ARGV[1]

-- 1. Safety Check: Ensure the idempotency key actually exists
if redis.call("EXISTS", idem_key) == 0 then 
    return 0 
end

-- 2. Update the forward map: Add job_id to the idempotency hash
redis.call("HSET", idem_key, "job_id", job_id)

-- 3. Create the reverse map: Store the idempotency key under the job_id
redis.call("SET", reverse_key, idem_key)

-- Mirror the primary key's remaining TTL onto the reverse mapping.
local ttl = redis.call("TTL", idem_key)
if ttl > 0 then
    redis.call("EXPIRE", reverse_key, ttl)
end

return 1 
`)
	idempotencyTransitionStatusScript = redis.NewScript(`
-- KEYS[1]: reverse mapping key (maps job ID to idempotency key)
-- ARGV[1]: new status
-- ARGV[2]: timestamp
-- ARGV[3]: extended TTL in seconds for completed state

local reverse_key = KEYS[1]
local new_status = ARGV[1]
local timestamp = ARGV[2]
local ttl = tonumber(ARGV[3])

local idem_key = redis.call("GET", reverse_key)

if not idem_key then
  return "missing"
end

local current_status = redis.call("HGET", idem_key, "status")

if not current_status then
  return "missing"
end

if current_status ~= "in_flight" then
  return "invalid"
end

redis.call("HSET", idem_key, "status", new_status, "completed_at", timestamp)
redis.call("EXPIRE", idem_key, ttl)
redis.call("EXPIRE", reverse_key, ttl)

return "ok"
`)
)

func (r *RedisStore) ClaimOrGet(
	ctx context.Context,
	userID string,
	idempotencyHeader string,
	ttl time.Duration,
) (*domain.IdempotencyResult, error) {

	idempotentKey := buildIdempotencyKey(userID, idempotencyHeader)

	now := time.Now().UTC().Format(time.RFC3339)

	raw, err := idempotencyScript.Run(
		ctx,
		r.redisClient,
		[]string{idempotentKey},
		int(ttl.Seconds()),
		now,
	).Result()
	if err != nil {
		return nil, fmt.Errorf("claim or get idempotency record: %w", err)
	}

	arr, ok := raw.([]interface{})
	if !ok || len(arr) != 4 {
		return nil, fmt.Errorf("unexpected lua result: %T", raw)
	}

	action := toString(arr[0])
	statusStr := toString(arr[1])
	createdAtStr := toString(arr[2])
	jobID := toString(arr[3])

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	expiresAt := createdAt.Add(ttl)

	switch action {
	case "NEW":
		return &domain.IdempotencyResult{
			Action:    domain.ActionNew,
			Status:    statusStr,
			JobID:     jobID,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			Err:       nil,
		}, nil

	case "DUPLICATE_IN_FLIGHT":
		return &domain.IdempotencyResult{
			Action:    domain.ActionDuplicateInFlight,
			Status:    statusStr,
			JobID:     jobID,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			Err:       fmt.Errorf("duplicated idempotency key"),
		}, nil

	case "REPLAY":
		return &domain.IdempotencyResult{
			Action:    domain.ActionReplay,
			Status:    statusStr,
			JobID:     jobID,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			Err:       fmt.Errorf("already processed idempotency key"),
		}, nil

	case "UNKNOWN_STATE":
		return &domain.IdempotencyResult{
			Action:    domain.ActionUnknown,
			Status:    statusStr,
			JobID:     jobID,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			Err:       fmt.Errorf("unknown state for idempotency key"),
		}, nil
	case "failed":
		return &domain.IdempotencyResult{
			Action:    domain.ActionFailed,
			Status:    statusStr,
			JobID:     jobID,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			Err:       fmt.Errorf("job has failed"),
		}, nil

	default:
		return nil, fmt.Errorf("unrecognized idempotency action: %q", action)
	}
}

func (r *RedisStore) SetJobID(ctx context.Context, userID, idempotencyHeader, jobID string) error {
	idempotentKey := buildIdempotencyKey(userID, idempotencyHeader)
	jobIDtoIdempotencyKey := fmt.Sprintf("jobID:%s:IdempotencyKey", jobID)

	result, err := setJobIDScript.Run(ctx, r.redisClient, []string{idempotentKey, jobIDtoIdempotencyKey}, jobID).Int()
	if err != nil {
		return fmt.Errorf("set idempotency job id: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("idempotency record not found for key=%s", idempotentKey)
	}
	return nil
}

func (r *RedisStore) Delete(ctx context.Context, userID, idempotencyHeader string) error {

	idempotentKey := buildIdempotencyKey(userID, idempotencyHeader)
	if err := r.redisClient.Del(ctx, idempotentKey).Err(); err != nil {
		return fmt.Errorf("delete idempotency key: %w", err)
	}
	return nil
}

func (r *RedisStore) TransitionStatus(ctx context.Context, userID, jobID, status string) error {

	jobIDtoIdempotencyKey := fmt.Sprintf("jobID:%s:IdempotencyKey", jobID)

	now := time.Now().UTC().Format(time.RFC3339)
	ttlSeconds := int(r.config.CompletedIdempotencyTTL.Seconds())
	result, err := idempotencyTransitionStatusScript.Run(ctx, r.redisClient, []string{jobIDtoIdempotencyKey},
		status,
		now,
		ttlSeconds).Result()
	if err != nil {
		return fmt.Errorf("transition idempotency status: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("cannot transition idempotency status for job=%s: %s", jobID, result)
	}
	return nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
