package usecases

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	authPublic "efficient-request-queueing-for-llm-inference/internal/auth/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeOAuth = "OAUTH_FAILED"

type AuthUseCase struct {
	oauthStateGenerator domain.OAuthStateGenerator
	oauthStateStore     domain.OAuthStateStore
	OAuthAuthorizer     domain.OAuthAuthorizer
	authTokenManager    domain.AuthTokenManager
	oauthRepo           domain.OAuthAccountRepository
	userRepo            domain.UserService
	uow                 domain.UnitOfWork
	logger              *slog.Logger
}

func NewAuthUseCase(
	oauthStateGenerator domain.OAuthStateGenerator,
	oauthStateStore domain.OAuthStateStore,
	oAuthAuthorizer domain.OAuthAuthorizer,
	authTokenManager domain.AuthTokenManager,
	oauthRepo domain.OAuthAccountRepository,
	userRepo domain.UserService,
	uow domain.UnitOfWork,
	logger *slog.Logger,
) *AuthUseCase {
	return &AuthUseCase{
		oauthStateGenerator: oauthStateGenerator,
		oauthStateStore:     oauthStateStore,
		OAuthAuthorizer:     oAuthAuthorizer,
		authTokenManager:    authTokenManager,
		oauthRepo:           oauthRepo,
		userRepo:            userRepo,
		uow:                 uow,
		logger:              logger,
	}
}

func (uc *AuthUseCase) GenerateAuthURL(ctx context.Context) (string, error) {
	state, err := uc.oauthStateGenerator.Generate()
	if err != nil {
		return "", sharederr.EnsureAppError(err, authPublic.ErrCodeStateNotGenerated, ErrTypeOAuth)

	}

	rdberr := uc.oauthStateStore.Store(ctx, state, "github", 5*time.Minute)
	if rdberr != nil {
		return "", sharederr.EnsureAppError(rdberr, sharederr.ErrCodeCacheWriteFailed, ErrTypeOAuth)
	}

	url := uc.OAuthAuthorizer.AuthorizationURL(state)
	return url, nil
}

func (uc *AuthUseCase) HandleCallback(ctx context.Context, code, state string) (*domain.AuthResponse, error) {
	err := uc.oauthStateStore.Exists(ctx, state, "github")
	if err != nil {
		return nil, sharederr.EnsureAppError(err, authPublic.ErrCodeUnauthorizedCallBack, ErrTypeOAuth)
	}

	err = uc.oauthStateStore.Delete(ctx, state, "github")
	if err != nil {
		uc.logger.WarnContext(
			ctx,
			"deleting the state param failed",
			"error", err,
		)

	}

	oauthAccessToken, err := uc.OAuthAuthorizer.ExchangeCode(ctx, code)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, authPublic.ErrCodeTokenExchangeFailed, ErrTypeOAuth)
	}

	oauthUser, err := uc.OAuthAuthorizer.GetUserInfo(ctx, oauthAccessToken)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, authPublic.ErrCodeFetchUserInfoFailed, ErrTypeOAuth)
	}

	authenticatedUser, err := uc.oauthRepo.FindByProviderUserID(ctx, oauthUser.Provider, oauthUser.ProviderUserID)
	if err != nil {
		if errors.Is(err, domain.ErrOauthAccountNotFound) {
			return uc.createNewUserAndLinkOAuth(ctx, oauthUser)
		}
		return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeOAuth)
	}

	role, err := uc.userRepo.Role(ctx, authenticatedUser.UserID)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeOAuth)
	}
	authenticatedUser.Role = role

	return uc.generateAuthResponse(authenticatedUser)
}

func (uc *AuthUseCase) createNewUserAndLinkOAuth(ctx context.Context, oauthUser *domain.OAuthUser) (*domain.AuthResponse, error) {

	result, err := uc.uow.Execute(ctx, func(txCtx context.Context) (string, error) {
		userUUID := uuid.New().String()

		if err := uc.userRepo.Register(txCtx, userUUID); err != nil {
			return "", err
		}

		res, err := uc.oauthRepo.Create(txCtx, oauthUser.ProviderUserID, oauthUser.Provider, userUUID)
		if err != nil {
			return "", err
		}

		return res.UserID, nil
	})

	if err != nil {
		return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeOAuth)
	}

	role, err := uc.userRepo.Role(ctx, result)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, sharederr.ErrCodeDBQueryFailed, ErrTypeOAuth)
	}

	authenticatedUser := &domain.AuthenticatedUser{
		UserID: result,
		Role:   role,
	}

	return uc.generateAuthResponse(authenticatedUser)
}

func (uc *AuthUseCase) generateAuthResponse(user *domain.AuthenticatedUser) (*domain.AuthResponse, error) {
	accessToken, err := uc.authTokenManager.Generate(user)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, authPublic.ErrCodeJWTTokenGenerationFailed, ErrTypeOAuth)
	}

	return &domain.AuthResponse{
		AccessToken: accessToken,
	}, nil
}
