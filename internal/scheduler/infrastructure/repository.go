package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/scheduler/domain"
	schedulerConfig "efficient-request-queueing-for-llm-inference/internal/scheduler/infrastructure/config"
)

const (
	queueKeyPrefix       = "queue:user:"
	idempotencyKeyPrefix = "queue_idempotency_key:"
	activeUsersKey       = "scheduler:active_users"
	processingJobsKey    = "processing_jobs"
	completedJobsKey     = "completed_jobs"
)

func buildQueuePerUserKey(userID string) string {
	return fmt.Sprintf("%s%s", queueKeyPrefix, userID)
}

func buildJobIdempotencyKey(userID, jobID string, attempt int) string {
	return fmt.Sprintf("%s%s:%s:%d", idempotencyKeyPrefix, userID, jobID, attempt)
}

type RedisSchedulerRepository struct {
	client *redis.Client
	lease  time.Duration
	config schedulerConfig.SchedulerConfig
}

func NewRedisSchedulerRepository(client *redis.Client, lease time.Duration, cfg schedulerConfig.SchedulerConfig) *RedisSchedulerRepository {
	return &RedisSchedulerRepository{
		client: client,
		lease:  lease,
		config: cfg,
	}
}

func (r *RedisSchedulerRepository) ClaimNextJob(ctx context.Context, workerID int) (*domain.FairDequeueResult, error) {
	val, err := fairDequeueScript.Run(
		ctx,
		r.client,
		[]string{activeUsersKey, processingJobsKey},
		r.lease.Milliseconds()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("fair dequeue failed: %w", err)
	}

	if val == nil {
		return nil, nil
	}

	arr, ok := val.([]interface{})
	if !ok || len(arr) != 2 {
		return nil, fmt.Errorf("fair dequeue: unexpected result type %T", val)
	}

	userID, ok1 := arr[0].(string)
	jobID, ok2 := arr[1].(string)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("fair dequeue: non-string elements in result")
	}

	if userID == "" || jobID == "" {
		return nil, fmt.Errorf("fair dequeue: empty userID or jobID")
	}

	return &domain.FairDequeueResult{
		UserID: userID,
		JobID:  jobID}, nil

}

func (r *RedisSchedulerRepository) PushLeft(ctx context.Context, entry domain.QueueEntry) (*domain.EnqueueResult, error) {

	userQueueKey := buildQueuePerUserKey(entry.UserID)
	idempotentKey := buildJobIdempotencyKey(entry.UserID, entry.JobID, entry.CurrentAttempt)

	result, err := enqueueScript.Run(
		ctx,
		r.client,
		[]string{idempotentKey, userQueueKey, activeUsersKey},
		entry.JobID,
		int(r.config.IdempotencyKeyTTL.Seconds()),
		r.config.QueueCapacity,
		entry.UserID).Result()
	if err != nil {
		return nil, fmt.Errorf("redis enqueue script: %w", err)
	}

	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 1 {
		return nil, fmt.Errorf("unexpected result format from redis enqueue script: got %T", result)
	}

	decision, ok := resultArray[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected success value type from redis enqueue script: got %T", resultArray[0])
	}

	switch decision {
	case 0:
		return nil, fmt.Errorf("%w", domain.ErrQueueFull)
	case 1:
		return nil, fmt.Errorf("%w", domain.ErrDuplicateJob)
	case 2:
		return &domain.EnqueueResult{
			BecameActive: true,
		}, nil
	case 3:
		return &domain.EnqueueResult{
			BecameActive: false,
		}, nil
	}

	return nil, fmt.Errorf("unexpected result from redis enqueue script: %v", decision)

}

func (r *RedisSchedulerRepository) ReleaseProcessingJob(ctx context.Context, jobID string) error {
	removed, err := r.client.ZRem(ctx, processingJobsKey, jobID).Result()
	if err != nil {
		return fmt.Errorf("release processing job: %w", err)
	}

	if removed == 0 {
		return fmt.Errorf("jobID not found")
	}

	return nil
}

func (r *RedisSchedulerRepository) ExtendProcessingLease(ctx context.Context, jobID string) error {
	updated, err := extendLeaseScript.Run(ctx, r.client, []string{processingJobsKey}, r.lease.Milliseconds(), jobID).Int64()
	if err != nil {
		return fmt.Errorf("extend processing lease: %w", err)
	}
	if updated == 0 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *RedisSchedulerRepository) MarkJobCompleted(ctx context.Context, jobID string) error {
	if err := r.client.SAdd(ctx, completedJobsKey, jobID).Err(); err != nil {
		return fmt.Errorf("mark completed job: %w", err)
	}
	return nil
}

func (r *RedisSchedulerRepository) ExpiredJobIDs(ctx context.Context, cutoffMs int64, limit int) ([]string, error) {

	ids, err := r.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     processingJobsKey,
		Start:   "-inf",
		Stop:    fmt.Sprintf("%d", cutoffMs),
		ByScore: true,
		Offset:  0,
		Count:   int64(limit),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("list expired processing jobs: %w", err)
	}
	return ids, nil
}

func (r *RedisSchedulerRepository) RemoveIfExpired(ctx context.Context, jobID string, cutoffMs int64) (bool, error) {
	n, err := removeExpiredScript.Run(ctx, r.client, []string{processingJobsKey}, jobID, cutoffMs).Int()
	if err != nil {
		return false, fmt.Errorf("remove expired processing job: %w", err)
	}
	return n == 1, nil
}

func (r *RedisSchedulerRepository) RequeueIfExpired(ctx context.Context, jobID, userID string, cutoffMs int64) (bool, error) {
	n, err := requeueExpiredScript.Run(ctx, r.client, []string{processingJobsKey, activeUsersKey, buildQueuePerUserKey(userID)}, jobID, userID, cutoffMs).Int()
	if err != nil {
		return false, fmt.Errorf("requeue expired processing job: %w", err)
	}
	return n == 1, nil
}

func (r *RedisSchedulerRepository) CompletedJobIDs(ctx context.Context) ([]string, error) {
	ids, err := r.client.SMembers(ctx, completedJobsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list completed jobs: %w", err)
	}
	return ids, nil
}

func (r *RedisSchedulerRepository) RemoveCompletedJob(ctx context.Context, jobID string) error {
	if err := r.client.SRem(ctx, completedJobsKey, jobID).Err(); err != nil {
		return fmt.Errorf("remove completed job marker: %w", err)
	}
	return nil
}

func (r *RedisSchedulerRepository) EnqueueIfAbsent(ctx context.Context, jobID, userID string, capacity int) (bool, error) {
	result, err := enqueueOrphanScript.Run(ctx, r.client, []string{completedJobsKey, processingJobsKey, buildQueuePerUserKey(userID), activeUsersKey}, jobID, userID, capacity).Int()
	if err != nil {
		return false, fmt.Errorf("enqueue orphan job: %w", err)
	}
	return result == 1, nil
}
