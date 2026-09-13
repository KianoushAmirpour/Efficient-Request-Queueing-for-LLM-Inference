package domain

import "context"

type UserService interface {
	Register(ctx context.Context, userUUID string) error
	Role(ctx context.Context, userUUID string) (string, error)
}
