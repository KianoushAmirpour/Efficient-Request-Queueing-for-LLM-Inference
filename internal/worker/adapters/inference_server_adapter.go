package adapters

import (
	"context"
	"fmt"

	inferenceServerPublicAPI "efficient-request-queueing-for-llm-inference/internal/inference_server/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type InferenceEngineAdapter struct {
	generator inferenceServerPublicAPI.StreamGenerator
}

var _ domain.InferenceEngine = InferenceEngineAdapter{}

func NewInferenceEngineAdapter(generator inferenceServerPublicAPI.StreamGenerator) InferenceEngineAdapter {
	return InferenceEngineAdapter{
		generator: generator,
	}
}

func (a InferenceEngineAdapter) GenerateStream(
	ctx context.Context,
	input domain.GenerationInput,
	onChunk func(domain.StreamChunk) error,
) error {

	genInput := inferenceServerPublicAPI.GenerationInput{
		Model:       input.Model,
		Prompt:      input.Prompt,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
	}

	err := a.generator.GenerateStream(
		ctx,
		genInput,
		func(chunk inferenceServerPublicAPI.GenerationChunk) error {
			return onChunk(domain.StreamChunk{Text: chunk.Text})
		},
	)
	if err != nil {
		if sharederr.IsCode(err, inferenceServerPublicAPI.ErrCodeServerOverloaded) {
			return fmt.Errorf("%w: %w", domain.ErrTransientInference, err)
		}
		return err
	}

	return nil
}
