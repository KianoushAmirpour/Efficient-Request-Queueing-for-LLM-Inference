package coalescing

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
)

type CoalescingService struct {
	redisClient *redis.Client
}

func NewCoalescingService(client *redis.Client) *CoalescingService {
	return &CoalescingService{
		redisClient: client,
	}
}

func buildCoalescingKey(userID string, hashRequest uint64) string {
	return fmt.Sprintf("coalescing:%s:inference:%d", userID, hashRequest)
}

var (
	coalescingScript = redis.NewScript(`
-- KEYS[1]: coalescing key
-- ARGV[1]: TTL in seconds
-- ARGV[2]: current timestamp

local key = KEYS[1]
local ttl = tonumber(ARGV[1])
local now = ARGV[2]

local created = redis.call("SET", key, now, "NX", "EX", ttl)

if created then
  return "PROCEED"
end

return "REJECTED"
`)
)

func (s *CoalescingService) TryCoalesce(ctx context.Context, userID string, hashRequest uint64, ttl time.Duration) (domain.CoalescingDecision, error) {

	coalescingKey := buildCoalescingKey(userID, hashRequest)

	now := time.Now().UTC().Format(time.RFC3339)

	result, err := coalescingScript.Run(
		ctx,
		s.redisClient,
		[]string{coalescingKey},
		int(ttl.Seconds()),
		now,
	).Result()
	if err != nil {
		return domain.CoalescingDecision{}, fmt.Errorf("coalescing failed: %w", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		return domain.CoalescingDecision{}, fmt.Errorf("invalid lua response: %T", result)
	}

	switch resultStr {
	case "PROCEED":
		return domain.CoalescingDecision{
			Action: domain.PROCEED,
			Error:  nil,
		}, nil
	case "REJECTED":
		return domain.CoalescingDecision{
			Action: domain.REJECTED,
			Error:  fmt.Errorf("coalescing rejected: %s", resultStr),
		}, nil
	default:
		return domain.CoalescingDecision{
			Action: domain.UNKNOWN,
			Error:  fmt.Errorf("unexpected lua action: %s", resultStr),
		}, nil
	}
}

func (s *CoalescingService) Delete(ctx context.Context, userID string, hashRequest uint64) error {

	coalescingKey := buildCoalescingKey(userID, hashRequest)

	if err := s.redisClient.Del(ctx, coalescingKey).Err(); err != nil {
		return fmt.Errorf("delete coalescing key: %w", err)
	}
	return nil
}
