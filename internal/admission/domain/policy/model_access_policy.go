package policy

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
)

type ModelAccessPolicy struct {
	policy domain.AdmissionPolicyConfig
}

func NewModelAccessPolicy(policy domain.AdmissionPolicyConfig) *ModelAccessPolicy {
	return &ModelAccessPolicy{
		policy: policy,
	}
}

func (p *ModelAccessPolicy) Evaluate(
	ctx context.Context,
	policyCtx *domain.PolicyEvaluationContext,
) error {

	userTier := policyCtx.User.Tier
	tierCfg := p.policy.Tiers[userTier]

	userModel := policyCtx.Request.Model
	_, ok := tierCfg.AllowedModels[userModel]

	if !ok {
		return domain.ErrModelNotAllowed
	}

	return nil
}
