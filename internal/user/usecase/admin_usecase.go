package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/user/domain"
	userPublic "efficient-request-queueing-for-llm-inference/internal/user/public"
)

const ErrTypeAdmin = "ADMIN_USE_CASE_FAILED"

type AdminUseCase struct {
	adminService     domain.AdminService
	tokenService     domain.AuthTokenManager
	cacheInvalidator domain.UserTierCache
	logger           *slog.Logger
}

func NewAdminUseCase(
	adminService domain.AdminService,
	tokenService domain.AuthTokenManager,
	cacheInvalidator domain.UserTierCache,
	logger *slog.Logger,
) *AdminUseCase {
	return &AdminUseCase{
		adminService:     adminService,
		tokenService:     tokenService,
		cacheInvalidator: cacheInvalidator,
		logger:           logger,
	}
}

type CreateAdminResponse struct {
	UserID string `json:"user_id"`
}

type AdminAuthLoginResponse struct {
	Token string `json:"token"`
}

func (uc *AdminUseCase) CreateAdmin(ctx context.Context) (*CreateAdminResponse, error) {

	adminUserID := uuid.New().String()

	err := uc.adminService.SetUserAsAdmin(ctx, adminUserID)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, userPublic.ErrCodeAdminCreationFailed, ErrTypeAdmin)
	}

	return &CreateAdminResponse{
		UserID: adminUserID,
	}, nil
}

func (uc *AdminUseCase) AdminAuthLogin(ctx context.Context, userID string) (*AdminAuthLoginResponse, error) {

	err := uc.adminService.IsUserAdmin(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return nil, sharederr.EnsureAppError(err, userPublic.ErrCodeAdminNotFound, ErrTypeAdmin)

		case errors.Is(err, domain.ErrUserNotAdmin):
			return nil, sharederr.EnsureAppError(err, userPublic.ErrCodeUserNotAdmin, ErrTypeAdmin)
		default:
			return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeAdmin)
		}
	}

	token, err := uc.tokenService.Generate(userID, string(domain.RoleAdmin))
	if err != nil {
		return nil, sharederr.EnsureAppError(err, userPublic.ErrCodeAdminTokenGenFailed, ErrTypeAdmin)
	}

	return &AdminAuthLoginResponse{
		Token: token,
	}, nil
}

func (uc *AdminUseCase) UpdateUserTier(ctx context.Context, req *domain.UpdateUserTierRequest) error {

	err := uc.adminService.UpdateUserTier(ctx, req.UserID, req.Tier)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return sharederr.EnsureAppError(err, userPublic.ErrCodeUserNotFound, ErrTypeAdmin)
		default:
			return sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeAdmin)
		}
	}

	if err = uc.cacheInvalidator.Invalidate(ctx, req.UserID); err != nil {
		uc.logger.WarnContext(
			ctx,
			"failed to invalidate the cache after updating user tier",
			"userID", req.UserID,
			"error", err)
	}

	return nil
}
