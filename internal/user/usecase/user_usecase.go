package usecase

import (
	"context"
	"log/slog"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/user/domain"
	"efficient-request-queueing-for-llm-inference/internal/user/public"
)

type UserUseCase struct {
	userRepo domain.UserRepository
	cacheSrv domain.UserTierCache
	logger   *slog.Logger
}

func NewUserService(
	userRepo domain.UserRepository,
	cacheService domain.UserTierCache,
	logger *slog.Logger) public.UserService {
	return UserUseCase{
		userRepo: userRepo,
		cacheSrv: cacheService,
		logger:   logger,
	}
}

const ErrTypeUser = "USER_OPERATION_FAILED"

func (u UserUseCase) Register(ctx context.Context, userUUID string) error {
	err := u.userRepo.Save(ctx, userUUID)
	if err != nil {
		return sharederr.NewAppError(public.ErrCodeRegisterFailed, ErrTypeUser, err)
	}
	return nil
}

func (u UserUseCase) Tier(ctx context.Context, userUUID string) (string, error) {

	cachedUserTier, _ := u.cacheSrv.Get(ctx, userUUID)
	switch cachedUserTier {
	case "":
		user, err := u.userRepo.FindByID(ctx, userUUID)
		if err != nil {
			return "", sharederr.NewAppError(public.ErrCodeTierFetchFailed, ErrTypeUser, err)
		}

		cacheErr := u.cacheSrv.Set(ctx, userUUID, user.Tier)
		u.logger.DebugContext(ctx,
			"failed to cache the user tier",
			"userID", userUUID,
			"error", cacheErr)

		return user.Tier, nil

	default:
		return cachedUserTier, nil
	}
}

func (u UserUseCase) Role(ctx context.Context, userUUID string) (string, error) {
	user, err := u.userRepo.FindByID(ctx, userUUID)
	if err != nil {
		return "", sharederr.NewAppError(public.ErrCodeRoleFetchFailed, ErrTypeUser, err)
	}
	return string(user.Role), nil
}
