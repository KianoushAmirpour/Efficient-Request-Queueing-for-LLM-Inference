package auth

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/github"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/persistence"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/security"
	"efficient-request-queueing-for-llm-inference/internal/auth/public"
	"efficient-request-queueing-for-llm-inference/internal/auth/transport"
	"efficient-request-queueing-for-llm-inference/internal/auth/usecases"
	"efficient-request-queueing-for-llm-inference/internal/shared/tx/postgres"
)

type Authdependency struct {
	userRepo            domain.UserService
	oauthStateGenerator domain.OAuthStateGenerator
	oauthStateStore     domain.OAuthStateStore
	oAuthAuthorizer     domain.OAuthAuthorizer
	authTokenManager    domain.AuthTokenManager
	oauthRepo           domain.OAuthAccountRepository
	uow                 domain.UnitOfWork
	logger              *slog.Logger
}

type AuthenticationDeps struct {
	PGPool      *pgxpool.Pool
	RedisClient *redis.Client
	UserService domain.UserService
}

type AuthConfig = config.AuthConfig

func buildAuthDeps(
	logger *slog.Logger,
	pgdb *pgxpool.Pool,
	rdb *redis.Client,
	config config.AuthConfig,
	userRepo domain.UserService,
) Authdependency {
	stateManager := security.NewDefaultStateTokenManager()
	stateStoreManager := persistence.NewRedisStateTokenManager(rdb)
	oauthClient := github.NewHTTPClient(config)
	authTokenManager := security.NewJWTService(config)
	oauthRepo := persistence.NewOauthRepository(pgdb)
	uow := postgres.NewUnitOfWork(pgdb, logger)

	return Authdependency{
		userRepo:            userRepo,
		oauthStateGenerator: stateManager,
		oauthStateStore:     stateStoreManager,
		oAuthAuthorizer:     oauthClient,
		authTokenManager:    authTokenManager,
		oauthRepo:           oauthRepo,
		uow:                 uow,
		logger:              logger,
	}
}

type Module struct {
	authHandler      *transport.AuthHandler
	AuthTokenManager public.AuthTokenManager
}

func NewAuthenticationModule(
	deps AuthenticationDeps,
	cfg AuthConfig,
	logger *slog.Logger,
) (*Module, error) {
	if deps.PGPool == nil {
		return nil, fmt.Errorf("authentication dependencies: postgres pool must not be nil")
	}
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("authentication dependencies: redis client must not be nil")
	}
	if deps.UserService == nil {
		return nil, fmt.Errorf("authentication dependencies: user service must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("authentication logger must not be nil")
	}
	if err := validateAuthConfig(cfg); err != nil {
		return nil, err
	}

	authModDeps := buildAuthDeps(logger, deps.PGPool, deps.RedisClient, cfg, deps.UserService)
	authUseCase := usecases.NewAuthUseCase(
		authModDeps.oauthStateGenerator,
		authModDeps.oauthStateStore,
		authModDeps.oAuthAuthorizer,
		authModDeps.authTokenManager,
		authModDeps.oauthRepo,
		authModDeps.userRepo,
		authModDeps.uow,
		logger,
	)

	externalAuthUseCase := usecases.NewAuthenticator(authModDeps.authTokenManager)

	return &Module{
		authHandler:      transport.NewAuthHandler(authUseCase, authModDeps.logger),
		AuthTokenManager: externalAuthUseCase,
	}, nil
}

func validateAuthConfig(cfg AuthConfig) error {
	if strings.TrimSpace(cfg.GitHub.GitHubClientID) == "" || strings.TrimSpace(cfg.GitHub.GitHubClientSecret) == "" {
		return fmt.Errorf("authentication configuration: GitHub client credentials must not be empty")
	}
	callbackURL, err := url.ParseRequestURI(cfg.GitHub.GitHubCallbackURL)
	if err != nil || callbackURL.Scheme == "" || callbackURL.Host == "" {
		return fmt.Errorf("authentication configuration: GitHub callback URL must be an absolute URL")
	}
	if strings.TrimSpace(cfg.JwtToken.JWTAccessSecret) == "" {
		return fmt.Errorf("authentication configuration: JWT access secret must not be empty")
	}
	if strings.TrimSpace(cfg.JwtToken.JWTIssuer) == "" || strings.TrimSpace(cfg.JwtToken.JWTAudience) == "" || strings.TrimSpace(cfg.JwtToken.JWTSubject) == "" {
		return fmt.Errorf("authentication configuration: JWT issuer, audience, and subject must not be empty")
	}
	if cfg.JwtToken.JWTAccessExpiry <= 0 {
		return fmt.Errorf("authentication configuration: JWT access expiry must be greater than zero")
	}
	if cfg.HttpClient.TransportDialTimeout <= 0 || cfg.HttpClient.TLSHandshakeTimeout <= 0 || cfg.HttpClient.ResponseHeaderTimeout <= 0 || cfg.HttpClient.ConnectionIdleTimeout <= 0 {
		return fmt.Errorf("authentication configuration: HTTP client timeouts must be greater than zero")
	}
	if cfg.HttpClient.MaxIdleConnections <= 0 || cfg.HttpClient.MaxConnsPerHost <= 0 {
		return fmt.Errorf("authentication configuration: HTTP client connection limits must be greater than zero")
	}
	if cfg.HttpClient.RetryCount < 0 || cfg.HttpClient.RetryBaseDelay <= 0 {
		return fmt.Errorf("authentication configuration: HTTP client retry count must be non-negative and base delay must be greater than zero")
	}
	return nil
}

func (m *Module) Name() string { return "Authentication Module" }

func (m *Module) RegisterRoutes(api *gin.RouterGroup) error {

	transport.RegisterAuthRoutes(api, m.authHandler)

	return nil
}
