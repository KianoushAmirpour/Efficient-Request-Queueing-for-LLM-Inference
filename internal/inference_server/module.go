package inferenceserver

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	config "efficient-request-queueing-for-llm-inference/internal/inference_server/infrastructure/config"
	"efficient-request-queueing-for-llm-inference/internal/inference_server/infrastructure/openai"
	"efficient-request-queueing-for-llm-inference/internal/inference_server/public"
	"efficient-request-queueing-for-llm-inference/internal/inference_server/usecase"
)

type InferenceEngineDeps struct{}

type InferenceEngineConfig = config.InferenceClient

type Module struct {
	InferenceEngine public.StreamGenerator
}

func NewInferenceEngineModule(
	deps InferenceEngineDeps,
	cfg InferenceEngineConfig,
	logger *slog.Logger,
) (*Module, error) {
	_ = deps
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("inference engine configuration: API key must not be empty")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("inference engine configuration: base URL must not be empty")
	}
	if cfg.MaxRetries < 0 {
		return nil, fmt.Errorf("inference engine configuration: max retries must not be negative")
	}
	positiveDurations := []struct {
		name  string
		value time.Duration
	}{
		{name: "dial timeout", value: cfg.DialTimeout},
		{name: "TLS handshake timeout", value: cfg.TLSHandshakeTimeout},
		{name: "response header timeout", value: cfg.ResponseHeaderTimeout},
		{name: "stream idle timeout", value: cfg.StreamIdleTimeout},
		{name: "stream total timeout", value: cfg.StreamTotalTimeout},
		{name: "connection idle timeout", value: cfg.ConnectionIdleTimeout},
	}
	for _, duration := range positiveDurations {
		if duration.value <= 0 {
			return nil, fmt.Errorf("inference engine configuration: %s must be positive", duration.name)
		}
	}
	if cfg.StreamIdleTimeout > cfg.StreamTotalTimeout {
		return nil, fmt.Errorf("inference engine configuration: stream idle timeout must not exceed stream total timeout")
	}
	if cfg.MaxIdleConnections <= 0 {
		return nil, fmt.Errorf("inference engine configuration: max idle connections must be positive")
	}
	if cfg.MaxIdleConnectionsHost <= 0 {
		return nil, fmt.Errorf("inference engine configuration: max idle connections per host must be positive")
	}
	if logger == nil {
		return nil, fmt.Errorf("inference engine logger must not be nil")
	}

	llmClient := openai.NewOpenAIClient(cfg)
	generator := usecase.NewStreamUseCase(llmClient)

	return &Module{
		InferenceEngine: generator,
	}, nil
}

func (m *Module) Name() string { return "Inference Engine" }
