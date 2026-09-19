package concurrency_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	scheduler "efficient-request-queueing-for-llm-inference/internal/scheduler"
	schedulerPublic "efficient-request-queueing-for-llm-inference/internal/scheduler/public"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

const (
	concurrentSchedulerJobs          = 120
	concurrentSchedulerUsers         = 6
	concurrentSchedulerWorkers       = 8
	concurrentSchedulerQueuePrefix   = "queue:user:"
	concurrentSchedulerProcessingKey = "processing_jobs"
)

func TestConcurrentSchedulersCannotDequeueSameJobTwice(t *testing.T) {
	client := testenv.NewRedisClient(t)
	module, err := scheduler.NewSchedulerModule(
		scheduler.SchedulerDeps{RedisClient: client, LeaseTimeout: time.Minute},
		scheduler.SchedulerConfig{QueueCapacity: 200, IdempotencyKeyTTL: time.Minute},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("construct scheduler module: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	for i := 0; i < concurrentSchedulerJobs; i++ {
		entry := schedulerPublic.QueueEntry{
			UserID:         fmt.Sprintf("user-%c", 'a'+rune(i%concurrentSchedulerUsers)),
			JobID:          fmt.Sprintf("job-%03d", i),
			CurrentAttempt: 1,
		}
		if _, err := module.QueueService.Enqueue(ctx, entry); err != nil {
			cancel()
			t.Fatalf("seed job %d: %v", i, err)
		}
	}
	cancel()

	claims := make(chan string, concurrentSchedulerJobs)
	errs := make(chan error, concurrentSchedulerWorkers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for workerID := 0; workerID < concurrentSchedulerWorkers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-start
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				claim, err := module.JobClaimerService.NextJob(ctx, workerID)
				cancel()
				if err != nil {
					errs <- err
					return
				}
				if claim == nil {
					return
				}
				claims <- claim.JobID
			}
		}(workerID)
	}
	close(start)
	wg.Wait()
	close(claims)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent scheduler error: %v", err)
		}
	}

	seen := make(map[string]struct{}, concurrentSchedulerJobs)
	for jobID := range claims {
		if _, exists := seen[jobID]; exists {
			t.Fatalf("job %q was dequeued twice", jobID)
		}
		seen[jobID] = struct{}{}
	}
	if len(seen) != concurrentSchedulerJobs {
		t.Fatalf("unique claims=%d; want %d", len(seen), concurrentSchedulerJobs)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	processing, err := client.ZCard(ctx, concurrentSchedulerProcessingKey).Result()
	cancel()
	if err != nil {
		t.Fatalf("read processing jobs: %v", err)
	}
	if processing != concurrentSchedulerJobs {
		t.Fatalf("processing jobs=%d; want %d", processing, concurrentSchedulerJobs)
	}

	for i := 0; i < concurrentSchedulerUsers; i++ {
		userID := fmt.Sprintf("user-%c", 'a'+rune(i))
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		queueLength, err := client.LLen(ctx, concurrentSchedulerQueuePrefix+userID).Result()
		cancel()
		if err != nil {
			t.Fatalf("read queue %s: %v", userID, err)
		}
		if queueLength != 0 {
			t.Fatalf("queue %s length=%d; want 0 after all claims", userID, queueLength)
		}
	}
}
