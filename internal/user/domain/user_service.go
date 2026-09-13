package domain

import "context"

type UserService interface {
	Register(ctx context.Context, userUUID string) error
	Tier(ctx context.Context, userUUID string) (string, error)
	Role(ctx context.Context, userUUID string) (string, error)
}

type UserRepository interface {
	Save(ctx context.Context, userUUID string) error
	FindByID(ctx context.Context, userUUID string) (*User, error)
}
