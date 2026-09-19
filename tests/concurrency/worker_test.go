package concurrency_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"

	"github.com/redis/go-redis/v9"
)

func newRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 13})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	return client
}

func newSchedulerModule(t *testing.T, client *redis.Client) *scheduler.Module {
	t.Helper()
	module, err := scheduler.NewSchedulerModule(scheduler.SchedulerDeps{RedisClient: client, LeaseTimeout: time.Minute}, scheduler.SchedulerConfig{QueueCapacity: 10, IdempotencyKeyTTL: time.Minute}, slog.Default())
	if err != nil {
		t.Fatalf("construct scheduler module: %v", err)
	}
	return module
}

func concurrentClaims(t *testing.T, claimers []schedulerPublic.JobClaimer, count int) []string {
	t.Helper()
	if len(claimers) == 0 {
		t.Fatal("at least one claimer is required")
	}
	start := make(chan struct{})
	claims := make(chan string, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			claim, err := claimers[id%len(claimers)].NextJob(ctx, id)
			if err != nil {
				errs <- err
			} else if claim != nil {
				claims <- claim.JobID
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
	}
	result := make([]string, 0, len(claims))
	for id := range claims {
		result = append(result, id)
	}
	return result
}

func TestConcurrentWorkersClaimOneJobExactlyOnce(t *testing.T) {
	m := newSchedulerModule(t, newRedisClient(t))
	ctx := context.Background()
	if _, err := m.QueueService.Enqueue(ctx, schedulerPublic.QueueEntry{JobID: "worker-job", UserID: "worker-user"}); err != nil {
		t.Fatal(err)
	}
	claims := concurrentClaims(t, []schedulerPublic.JobClaimer{m.JobClaimerService}, 32)
	if len(claims) != 1 || claims[0] != "worker-job" {
		t.Fatalf("claims=%v want one worker-job claim", claims)
	}
}

func TestCompetingWorkerPoolsClaimOneJobExactlyOnce(t *testing.T) {
	client := newRedisClient(t)
	poolA := newSchedulerModule(t, client)
	poolB := newSchedulerModule(t, client)
	ctx := context.Background()
	if _, err := poolA.QueueService.Enqueue(ctx, schedulerPublic.QueueEntry{JobID: "pool-job", UserID: "pool-user"}); err != nil {
		t.Fatal(err)
	}
	claims := concurrentClaims(t, []schedulerPublic.JobClaimer{
		poolA.JobClaimerService,
		poolB.JobClaimerService,
	}, 32)
	if len(claims) != 1 || claims[0] != "pool-job" {
		t.Fatalf("claims=%v want one pool-job claim", claims)
	}
}
