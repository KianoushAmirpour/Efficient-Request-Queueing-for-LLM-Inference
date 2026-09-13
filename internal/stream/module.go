package stream

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
	redisinfra "efficient-request-queueing-for-llm-inference/internal/stream/infrastructure/redis"
	"efficient-request-queueing-for-llm-inference/internal/stream/ports"
	"efficient-request-queueing-for-llm-inference/internal/stream/public"
	"efficient-request-queueing-for-llm-inference/internal/stream/transport"
	"efficient-request-queueing-for-llm-inference/internal/stream/usecase"
)

type StreamDeps struct {
	RedisClient            *goredis.Client
	TokenValidationService ports.AccessTokenValidator
}

type StreamConfig struct{}

type streamDependency struct {
	publisher       *usecase.StreamPublisherUseCase
	subscriber      domain.Subscriber
	accessValidator ports.AccessTokenValidator
	logger          *slog.Logger
}

func buildStreamDeps(redisClient *goredis.Client, tokenValidator ports.AccessTokenValidator, logger *slog.Logger) streamDependency {
	streamService := redisinfra.NewStreamService(redisClient)

	publisher := usecase.NewStreamPublisherUseCase(streamService)

	return streamDependency{
		publisher:       publisher,
		subscriber:      streamService,
		accessValidator: tokenValidator,
		logger:          logger,
	}
}

type Module struct {
	streamHandler *transport.StreamHTTPHandler
	publisher     public.StreamPublisher
}

func NewStreamModule(deps StreamDeps, cfg StreamConfig, logger *slog.Logger) (*Module, error) {
	_ = cfg
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("stream dependencies: redis client must not be nil")
	}
	if deps.TokenValidationService == nil {
		return nil, fmt.Errorf("stream dependencies: token validation service must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("stream logger must not be nil")
	}

	moduleDeps := buildStreamDeps(deps.RedisClient, deps.TokenValidationService, logger)

	httpHandler := transport.NewStreamHTTPHandler(moduleDeps.subscriber, moduleDeps.accessValidator, logger)

	return &Module{
		streamHandler: httpHandler,
		publisher:     moduleDeps.publisher,
	}, nil
}

func (m *Module) Name() string {
	return "Stream"
}

func (m *Module) Publisher() public.StreamPublisher {
	return m.publisher
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup) error {
	transport.RegisterStreamRoutes(api, m.streamHandler)
	return nil
}
