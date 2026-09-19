package admission_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	admission "efficient-request-queueing-for-llm-inference/internal/admission"
	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	admissionConfig "efficient-request-queueing-for-llm-inference/internal/admission/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/tests/testenv"
)

const (
	freeUser       = "free-user"
	secondFreeUser = "second-free-user"
	premiumUser    = "premium-user"
	freeModel      = "free-model"
	premiumModel   = "premium-model"
)

type users struct {
	policies map[string]domain.UserPolicy
}

func (u users) GetUserPolicy(_ context.Context, userID string) (domain.UserPolicy, error) {
	policy, ok := u.policies[userID]
	if !ok {
		return domain.UserPolicy{}, fmt.Errorf("unknown user %q", userID)
	}
	return policy, nil
}

func newService(t *testing.T, client *redis.Client, freeCapacity, premiumCapacity, refillRate int) public.Admitter {
	t.Helper()

	module, err := admission.NewAdmissionModule(
		admission.AdmissionDeps{
			RedisClient: client,
			UserPolicyReader: users{policies: map[string]domain.UserPolicy{
				freeUser:       {Tier: "free"},
				secondFreeUser: {Tier: "free"},
				premiumUser:    {Tier: "premium"},
			}},
		},
		admission.AdmissionConfig{
			RateLimit: admissionConfig.RateLimitConfig{
				Free:    admissionConfig.RateLimit{Capacity: freeCapacity, RefillRate: refillRate, Ttl: time.Minute},
				Premium: admissionConfig.RateLimit{Capacity: premiumCapacity, RefillRate: refillRate, Ttl: time.Minute},
			},
			Tiers: map[string]admissionConfig.TierConfig{
				"free": {
					MaxInputTokens: 4, MaxOutputTokens: 2,
					Models: []admissionConfig.ModelInfo{{Name: freeModel, ContextLength: 32}},
				},
				"premium": {
					MaxInputTokens: 8, MaxOutputTokens: 3,
					Models: []admissionConfig.ModelInfo{{Name: premiumModel, ContextLength: 64}},
				},
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

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func admit(t *testing.T, service public.Admitter, userID, model string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := service.Admit(ctx, public.AdmitInput{UserID: userID, Model: model, Prompt: "hello"})
	return err
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !sharederr.IsCode(err, code) {
		t.Fatalf("error = %v, want application code %q", err, code)
	}
}

func TestAdmissionRateLimitExhaustionAndRefill(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newService(t, client, 6, 6, 6)

	for i := 0; i < 2; i++ {
		if err := admit(t, service, freeUser, freeModel); err != nil {
			t.Fatalf("request %d rejected before exhaustion: %v", i+1, err)
		}
	}
	requireCode(t, admit(t, service, freeUser, freeModel), public.ErrCodeRateLimited)

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := admit(t, service, freeUser, freeModel); err == nil {
			return
		} else if !sharederr.IsCode(err, public.ErrCodeRateLimited) {
			t.Fatalf("unexpected refill error: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("rate limiter did not refill within the expected interval")
}

func TestAdmissionRateLimitIsolatedPerUser(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newService(t, client, 3, 3, 1)

	if err := admit(t, service, freeUser, freeModel); err != nil {
		t.Fatalf("first user request rejected: %v", err)
	}
	requireCode(t, admit(t, service, freeUser, freeModel), public.ErrCodeRateLimited)

	if err := admit(t, service, secondFreeUser, freeModel); err != nil {
		t.Fatalf("second free-tier user should have an independent bucket: %v", err)
	}
}

func TestAdmissionRateLimitUsesTierSpecificCapacity(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newService(t, client, 3, 8, 1)

	if err := admit(t, service, freeUser, freeModel); err != nil {
		t.Fatalf("free request rejected: %v", err)
	}
	requireCode(t, admit(t, service, freeUser, freeModel), public.ErrCodeRateLimited)

	for i := 0; i < 2; i++ {
		if err := admit(t, service, premiumUser, premiumModel); err != nil {
			t.Fatalf("premium request %d rejected: %v", i+1, err)
		}
	}
	requireCode(t, admit(t, service, premiumUser, premiumModel), public.ErrCodeRateLimited)
}

func TestAdmissionRateLimiterRedisUnavailable(t *testing.T) {
	client := testenv.NewRedisClient(t)
	service := newService(t, client, 100, 100, 1)
	if err := client.Close(); err != nil {
		t.Fatalf("close Redis client: %v", err)
	}

	err := admit(t, service, freeUser, freeModel)
	if err == nil {
		t.Fatal("expected admission to fail when Redis is unavailable")
	}
	if !sharederr.IsCode(err, sharederr.ErrCodeInternal) {
		t.Fatalf("error = %v, want code %q", err, sharederr.ErrCodeInternal)
	}
}
