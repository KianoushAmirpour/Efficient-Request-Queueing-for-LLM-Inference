package concurrency_test

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	admission "efficient-request-queueing-for-llm-inference/internal/admission"
	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	admissionConfig "efficient-request-queueing-for-llm-inference/internal/admission/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
	"github.com/redis/go-redis/v9"
)

type concurrencyUsers struct{}

func (concurrencyUsers) GetUserPolicy(context.Context, string) (domain.UserPolicy, error) {
	return domain.UserPolicy{Tier: "free"}, nil
}

func newConcurrencyService(t *testing.T, client *redis.Client) public.Admitter {
	t.Helper()
	module, err := admission.NewAdmissionModule(
		admission.AdmissionDeps{RedisClient: client, UserPolicyReader: concurrencyUsers{}},
		admission.AdmissionConfig{
			RateLimit: admissionConfig.RateLimitConfig{
				Free:    admissionConfig.RateLimit{Capacity: 5, RefillRate: 1, Ttl: time.Minute},
				Premium: admissionConfig.RateLimit{Capacity: 5, RefillRate: 1, Ttl: time.Minute},
			},
			Tiers: map[string]admissionConfig.TierConfig{
				"free": {MaxInputTokens: 4, MaxOutputTokens: 1, Models: []admissionConfig.ModelInfo{{Name: "toy-model", ContextLength: 32}}},
			},
		},
		slog.New(slog.NewTextHandler(testWriter{t}, nil)),
	)
	if err != nil {
		t.Fatalf("construct admission module: %v", err)
	}
	return module.AdmissionService
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func TestRateLimitAtomicConcurrentConsumption(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newConcurrencyService(t, client)

	const requests = 20
	var allowed, rateLimited, unexpected atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := service.Admit(ctx, public.AdmitInput{UserID: "toy-user", Model: "toy-model", Prompt: "toy"})
			switch {
			case err == nil:
				allowed.Add(1)
			case sharederr.IsCode(err, public.ErrCodeRateLimited):
				rateLimited.Add(1)
			default:
				unexpected.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := allowed.Load(); got != 5 {
		t.Fatalf("allowed concurrent requests = %d, want exactly 5", got)
	}
	if got := rateLimited.Load(); got != requests-5 {
		t.Fatalf("rate-limited concurrent requests = %d, want %d", got, requests-5)
	}
	if got := unexpected.Load(); got != 0 {
		t.Fatalf("unexpected concurrent admission errors = %d", got)
	}
	if got := allowed.Load() + rateLimited.Load(); got != requests {
		t.Fatalf("request accounting mismatch: accounted=%d, want %d", got, requests)
	}
}
