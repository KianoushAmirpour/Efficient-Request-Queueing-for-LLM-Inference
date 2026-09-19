package admission_test

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	inferenceAdapters "efficient-request-queueing-for-llm-inference/internal/inference/adapters"
	"efficient-request-queueing-for-llm-inference/internal/inference/domain"
	inferenceConfig "efficient-request-queueing-for-llm-inference/internal/inference/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/inference/ports"
	inferenceUsecase "efficient-request-queueing-for-llm-inference/internal/inference/usecase"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

type downstreamCounters struct {
	created  atomic.Int32
	enqueued atomic.Int32
}

func (d *downstreamCounters) Create(context.Context, *domain.InferenceRequest) (*domain.InferenceJob, error) {
	d.created.Add(1)
	return &domain.InferenceJob{JobID: "unexpected-job", UserID: "free-user", CurrentAttempt: 1}, nil
}

func (*downstreamCounters) MarkFailed(context.Context, string, int) error { return nil }

func (d *downstreamCounters) TryEnqueue(context.Context, domain.InferenceJob) (bool, error) {
	d.enqueued.Add(1)
	return false, nil
}

type fixedIdempotency struct{}

func (fixedIdempotency) ClaimOrGet(context.Context, string, string, time.Duration) (*domain.IdempotencyResult, error) {
	return &domain.IdempotencyResult{Action: domain.ActionNew}, nil
}
func (fixedIdempotency) SetJobID(context.Context, string, string, string) error { return nil }
func (fixedIdempotency) Delete(context.Context, string, string) error           { return nil }
func (fixedIdempotency) TransitionStatus(context.Context, string, string) error { return nil }

type fixedCoalescer struct{}

func (fixedCoalescer) TryCoalesce(context.Context, string, uint64, time.Duration) (domain.CoalescingDecision, error) {
	return domain.CoalescingDecision{Action: domain.PROCEED}, nil
}
func (fixedCoalescer) Delete(context.Context, string, uint64) error { return nil }

type fixedHasher struct{}

func (fixedHasher) HashRequest(string) uint64 { return 1 }

var _ ports.JobService = (*downstreamCounters)(nil)
var _ ports.Enqueuer = (*downstreamCounters)(nil)
var _ ports.IdempotencyStore = fixedIdempotency{}
var _ ports.Coalescer = fixedCoalescer{}
var _ ports.Hasher = fixedHasher{}

func TestAdmissionRejectsBeforeDownstreamWork(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newService(t, client, 100, 100, 1)
	counters := &downstreamCounters{}

	usecase := inferenceUsecase.NewInferenceUseCase(
		inferenceAdapters.NewRequestAdmitterAdapter(service),
		fixedIdempotency{},
		fixedHasher{},
		fixedCoalescer{},
		counters,
		counters,
		inferenceConfig.InferenceConfig{
			CoalescingTTL: time.Minute, IdempotencyTTL: time.Minute, CompletedIdempotencyTTL: time.Hour,
		},
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)

	_, err := usecase.Submit(context.Background(), &domain.InferenceInput{
		Prompt: "hello",
		Model:  premiumModel,
	}, freeUser, "request-1")
	requireCode(t, err, public.ErrCodeModelNotAllowed)
	if got := counters.created.Load(); got != 0 {
		t.Fatalf("job creation count = %d, want 0", got)
	}
	if got := counters.enqueued.Load(); got != 0 {
		t.Fatalf("enqueue count = %d, want 0", got)
	}
	if sharederr.Code(err) != public.ErrCodeModelNotAllowed {
		t.Fatalf("rejection code = %q, want %q", sharederr.Code(err), public.ErrCodeModelNotAllowed)
	}
}
