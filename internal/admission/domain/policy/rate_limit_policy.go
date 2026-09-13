package policy

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
)

type RateLimitPolicy struct {
	limiter domain.Limiter
}

func NewRateLimitPolicy(
	limiter domain.Limiter,
) *RateLimitPolicy {
	return &RateLimitPolicy{
		limiter: limiter,
	}
}

func (p *RateLimitPolicy) Evaluate(
	ctx context.Context,
	policyCtx *domain.PolicyEvaluationContext,
) error {

	err := p.limiter.Allow(
		ctx,
		policyCtx.Request.UserID,
		policyCtx.User.Tier,
		policyCtx.Metadata.InputTokens,
		policyCtx.Metadata.MaxOutputTokens,
	)
	if err != nil {
		return err
	}

	return nil
}
