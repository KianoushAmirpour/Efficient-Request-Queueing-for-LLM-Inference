package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	"efficient-request-queueing-for-llm-inference/internal/user/public"
)

type UserAdapter struct {
	UserService public.UserService
}

var _ domain.UserService = UserAdapter{}

func NewUserAdapter(userService public.UserService) UserAdapter {
	return UserAdapter{UserService: userService}
}

func (u UserAdapter) Register(ctx context.Context, userUUID string) error {
	return u.UserService.Register(ctx, userUUID)
}

func (u UserAdapter) Role(ctx context.Context, userUUID string) (string, error) {
	return u.UserService.Role(ctx, userUUID)
}
