package policy

import (
	"context"
	"unicode/utf8"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
)

type TokenLimitPolicy struct {
	policy domain.AdmissionPolicyConfig
}

func NewTokenLimitPolicy(policy domain.AdmissionPolicyConfig) *TokenLimitPolicy {
	return &TokenLimitPolicy{
		policy: policy,
	}
}

func (p *TokenLimitPolicy) Evaluate(
	ctx context.Context,
	policyCtx *domain.PolicyEvaluationContext,
) error {

	estimatedTokens := utf8.RuneCountInString(policyCtx.Request.Prompt) / 4

	userTier := policyCtx.User.Tier
	pl := p.policy.Tiers[userTier]

	if estimatedTokens > pl.MaxInputTokens {
		return domain.ErrInputTooLarge
	}

	contextLength, ok := pl.ModelContextLengths[policyCtx.Request.Model]
	if ok && estimatedTokens+pl.MaxOutputTokens > contextLength {
		return domain.ErrContextLengthExceeded
	}

	policyCtx.Metadata.InputTokens = estimatedTokens
	policyCtx.Metadata.MaxOutputTokens = pl.MaxOutputTokens

	return nil
}
