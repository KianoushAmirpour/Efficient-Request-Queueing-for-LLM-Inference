package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	"efficient-request-queueing-for-llm-inference/internal/admission/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type admissionTestUsers struct {
	policies map[string]domain.UserPolicy
	err      error
}

func (f admissionTestUsers) GetUserPolicy(_ context.Context, userID string) (domain.UserPolicy, error) {
	if f.err != nil {
		return domain.UserPolicy{}, f.err
	}

	policy, ok := f.policies[userID]
	if !ok {
		return domain.UserPolicy{}, errors.New("user policy not found")
	}

	return policy, nil
}

type admissionTestPolicy struct {
	err       error
	evaluates int
	metadata  domain.RequestMetadata
	setMeta   bool
}

func (f *admissionTestPolicy) Evaluate(_ context.Context, policyCtx *domain.PolicyEvaluationContext) error {
	f.evaluates++
	if f.err != nil {
		return f.err
	}

	if f.setMeta {
		policyCtx.Metadata = f.metadata
	}
	return nil
}

func TestAdmissionUseCaseAdmitsValidFreeAndPremiumRequests(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		model           string
		tier            string
		maxOutputTokens int
	}{
		{
			name:            "free tier",
			userID:          "free-user",
			model:           "small-model",
			tier:            "free",
			maxOutputTokens: 128,
		},
		{
			name:            "premium tier",
			userID:          "premium-user",
			model:           "large-model",
			tier:            "premium",
			maxOutputTokens: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelPolicy := &admissionTestPolicy{}
			tokenPolicy := &admissionTestPolicy{
				metadata: domain.RequestMetadata{
					InputTokens:     12,
					MaxOutputTokens: tt.maxOutputTokens,
				},
				setMeta: true,
			}
			rateLimitPolicy := &admissionTestPolicy{}
			service := NewAdmissionUseCase(
				admissionTestUsers{
					policies: map[string]domain.UserPolicy{
						tt.userID: {Tier: tt.tier},
					},
				},
				tokenPolicy,
				modelPolicy,
				rateLimitPolicy,
			)

			input := public.AdmitInput{
				UserID: tt.userID,
				Prompt: "a valid prompt",
				Model:  tt.model,
			}
			got, err := service.Admit(context.Background(), input)
			if err != nil {
				t.Fatalf("Admit() error = %v", err)
			}

			want := public.AdmitOutput{
				Admitted:        true,
				UserID:          input.UserID,
				Prompt:          input.Prompt,
				Model:           input.Model,
				MaxOutputTokens: tt.maxOutputTokens,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Admit() output = %#v, want %#v", got, want)
			}
			if modelPolicy.evaluates != 1 || tokenPolicy.evaluates != 1 || rateLimitPolicy.evaluates != 1 {
				t.Fatalf("policy evaluations = model:%d token:%d rate-limit:%d, want one each", modelPolicy.evaluates, tokenPolicy.evaluates, rateLimitPolicy.evaluates)
			}
		})
	}
}

func TestAdmissionUseCaseRejectsModelNotAllowedForTier(t *testing.T) {
	modelPolicy := &admissionTestPolicy{err: domain.ErrModelNotAllowed}
	tokenPolicy := &admissionTestPolicy{}
	rateLimitPolicy := &admissionTestPolicy{}
	service := NewAdmissionUseCase(
		admissionTestUsers{policies: map[string]domain.UserPolicy{"free-user": {Tier: "free"}}},
		tokenPolicy,
		modelPolicy,
		rateLimitPolicy,
	)

	got, err := service.Admit(context.Background(), public.AdmitInput{
		UserID: "free-user",
		Prompt: "a valid prompt",
		Model:  "premium-only-model",
	})
	if err == nil || !sharederr.IsCode(err, public.ErrCodeModelNotAllowed) {
		t.Fatalf("Admit() error = %v, want code %q", err, public.ErrCodeModelNotAllowed)
	}
	if !reflect.DeepEqual(got, public.AdmitOutput{}) {
		t.Fatalf("Admit() output = %#v, want zero output", got)
	}
	if tokenPolicy.evaluates != 0 || rateLimitPolicy.evaluates != 0 {
		t.Fatalf("later policies evaluated: token=%d rate-limit=%d", tokenPolicy.evaluates, rateLimitPolicy.evaluates)
	}
}

func TestAdmissionUseCaseRejectsInputExceedingTierLimit(t *testing.T) {
	modelPolicy := &admissionTestPolicy{}
	tokenPolicy := &admissionTestPolicy{err: domain.ErrInputTooLarge}
	rateLimitPolicy := &admissionTestPolicy{}
	service := NewAdmissionUseCase(
		admissionTestUsers{policies: map[string]domain.UserPolicy{"free-user": {Tier: "free"}}},
		tokenPolicy,
		modelPolicy,
		rateLimitPolicy,
	)

	got, err := service.Admit(context.Background(), public.AdmitInput{
		UserID: "free-user",
		Prompt: "an input beyond the free tier limit",
		Model:  "small-model",
	})
	if err == nil || !sharederr.IsCode(err, public.ErrCodeInputToken) {
		t.Fatalf("Admit() error = %v, want code %q", err, public.ErrCodeInputToken)
	}
	if !reflect.DeepEqual(got, public.AdmitOutput{}) {
		t.Fatalf("Admit() output = %#v, want zero output", got)
	}
	if rateLimitPolicy.evaluates != 0 {
		t.Fatalf("rate-limit policy evaluated %d times, want zero", rateLimitPolicy.evaluates)
	}
}

func TestAdmissionUseCaseRejectsUnknownUserPolicyFailure(t *testing.T) {
	modelPolicy := &admissionTestPolicy{}
	tokenPolicy := &admissionTestPolicy{}
	rateLimitPolicy := &admissionTestPolicy{}
	service := NewAdmissionUseCase(
		admissionTestUsers{policies: map[string]domain.UserPolicy{}},
		tokenPolicy,
		modelPolicy,
		rateLimitPolicy,
	)

	got, err := service.Admit(context.Background(), public.AdmitInput{
		UserID: "unknown-user",
		Prompt: "a valid prompt",
		Model:  "small-model",
	})
	if err == nil || !sharederr.IsCode(err, public.ErrCodePolicyFetchFailed) {
		t.Fatalf("Admit() error = %v, want code %q", err, public.ErrCodePolicyFetchFailed)
	}
	if !reflect.DeepEqual(got, public.AdmitOutput{}) {
		t.Fatalf("Admit() output = %#v, want zero output", got)
	}
	if modelPolicy.evaluates != 0 || tokenPolicy.evaluates != 0 || rateLimitPolicy.evaluates != 0 {
		t.Fatalf("policies evaluated after user lookup failure: model=%d token=%d rate-limit=%d", modelPolicy.evaluates, tokenPolicy.evaluates, rateLimitPolicy.evaluates)
	}
}
