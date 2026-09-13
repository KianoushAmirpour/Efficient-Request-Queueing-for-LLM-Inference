package inferenceserver

import (
	"fmt"
	"log/slog"

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
	if logger == nil {
		return nil, fmt.Errorf("inference engine logger must not be nil")
	}

	llmClient := openai.NewOpenAIClient(cfg.APIKey)
	generator := usecase.NewStreamUseCase(llmClient)

	return &Module{
		InferenceEngine: generator,
	}, nil
}

func (m *Module) Name() string { return "Inference Engine" }
