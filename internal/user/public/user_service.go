package public

import "context"

type UserService interface {
	Register(ctx context.Context, userUUID string) error       // return USER_REGISTRATION_FAILED as ErrCode
	Tier(ctx context.Context, userUUID string) (string, error) // return USER_TIER_FETCH_FAILED as ErrCode
	Role(ctx context.Context, userUUID string) (string, error) // return USER_ROLE_FETCH_FAILED as ErrCode
}
