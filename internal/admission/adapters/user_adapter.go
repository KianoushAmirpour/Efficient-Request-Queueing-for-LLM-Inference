package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/admission/domain"
	userPublicAPI "efficient-request-queueing-for-llm-inference/internal/user/public"
)

type UserPolicyAdapter struct {
	UserService userPublicAPI.UserService
}

var _ domain.UserPolicyReader = UserPolicyAdapter{}

func NewUserPolicyAdapter(userService userPublicAPI.UserService) UserPolicyAdapter {
	return UserPolicyAdapter{UserService: userService}
}

func (a UserPolicyAdapter) GetUserPolicy(ctx context.Context, userID string) (domain.UserPolicy, error) {
	userTier, err := a.UserService.Tier(ctx, userID)
	if err != nil {
		return domain.UserPolicy{}, err
	}

	return domain.UserPolicy{
		Tier: userTier,
	}, nil
}
