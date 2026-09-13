package domain

import (
	"context"
)

type UserPolicyReader interface {
	GetUserPolicy(ctx context.Context, userID string) (UserPolicy, error)
}

type Limiter interface {
	Allow(ctx context.Context, userID string, tier string, inputTokens, maxOutpuTokens int) error
}

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, policyCtx *PolicyEvaluationContext) error
}
