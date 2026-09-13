package user

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/user/domain"
	"efficient-request-queueing-for-llm-inference/internal/user/infrastructure/cache"
	"efficient-request-queueing-for-llm-inference/internal/user/infrastructure/persistence"
	"efficient-request-queueing-for-llm-inference/internal/user/transport"
	"efficient-request-queueing-for-llm-inference/internal/user/usecase"
)

type userDependency struct {
	UserService  domain.UserService
	AdminService domain.AdminService
	CacheService domain.UserTierCache
	logger       *slog.Logger
}

type UserDeps struct {
	PGPool      *pgxpool.Pool
	RedisClient *redis.Client
}

type UserConfig struct{}

func buildUserDeps(pgdb *pgxpool.Pool, redisClient *redis.Client, logger *slog.Logger) userDependency {

	userRepo := persistence.NewUserRepository(pgdb)
	cacheSvc := cache.NewCacher(redisClient)
	userSvc := usecase.NewUserService(userRepo, cacheSvc, logger)

	return userDependency{
		AdminService: userRepo,
		UserService:  userSvc,
		CacheService: cacheSvc,
		logger:       logger,
	}
}

type Module struct {
	tokenService domain.AuthTokenManager
	adminHandler *transport.AdminHandler
	UserService  domain.UserService
	userDeps     userDependency
}

func NewUserModule(deps UserDeps, cfg UserConfig, logger *slog.Logger) (*Module, error) {
	_ = cfg
	if deps.PGPool == nil {
		return nil, fmt.Errorf("user dependencies: postgres pool must not be nil")
	}
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("user dependencies: redis client must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("user logger must not be nil")
	}

	userModDeps := buildUserDeps(deps.PGPool, deps.RedisClient, logger)

	return &Module{
		UserService: userModDeps.UserService,
		userDeps:    userModDeps,
	}, nil
}

func (m *Module) SetTokenService(tokenService domain.AuthTokenManager) error {
	if tokenService == nil {
		return fmt.Errorf("tokenService cannot be nil")
	}
	m.tokenService = tokenService
	adminUseCase := usecase.NewAdminUseCase(m.userDeps.AdminService, tokenService, m.userDeps.CacheService, m.userDeps.logger)
	m.adminHandler = transport.NewAdminHandler(adminUseCase, m.userDeps.logger)

	return nil
}

func (m *Module) Name() string { return "User" }

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) error {
	if m.tokenService == nil || m.adminHandler == nil {
		panic("token service and admin handler must be set")
	}
	transport.RegisterRoutes(rg, m.adminHandler, m.tokenService)
	return nil
}
