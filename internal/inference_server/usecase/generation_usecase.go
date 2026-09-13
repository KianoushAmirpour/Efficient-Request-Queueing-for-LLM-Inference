package usecase

import (
	"context"
	"errors"

	"efficient-request-queueing-for-llm-inference/internal/inference_server/domain"
	"efficient-request-queueing-for-llm-inference/internal/inference_server/public"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

const ErrTypeInferenceServer = "INFERENCE_SERVER_FAILED"

type streamUseCase struct {
	client domain.LLMClient
}

func NewStreamUseCase(llmClient domain.LLMClient) public.StreamGenerator {
	return &streamUseCase{client: llmClient}
}

func (u *streamUseCase) GenerateStream(
	ctx context.Context,
	input public.GenerationInput,
	onChunk func(public.GenerationChunk) error,
) error {

	req := domain.GenerationRequest{
		Model:       input.Model,
		Prompt:      input.Prompt,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
	}

	err := u.client.GenerateStream(ctx, req, func(chunk domain.GenerationChunk) error {

		pubChunk := public.GenerationChunk{Text: chunk.Text}
		if onChunk != nil {
			return onChunk(pubChunk)
		}
		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrInvalidRequest):
		return sharederr.EnsureAppError(err, public.ErrCodeInvalidRequest, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrAuthenticationFailed):
		return sharederr.EnsureAppError(err, public.ErrCodeAuthenticationFailed, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrModelNotFound):
		return sharederr.EnsureAppError(err, public.ErrCodeModelNotFound, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrContentFiltered):
		return sharederr.EnsureAppError(err, public.ErrCodeContentFiltered, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrMaxTokensExceeded):
		return sharederr.EnsureAppError(err, public.ErrCodeMaxTokensExceeded, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrServerOverloaded):
		return sharederr.EnsureAppError(err, public.ErrCodeServerOverloaded, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrRateLimitExceeded):
		return sharederr.EnsureAppError(err, public.ErrCodeRateLimitExceeded, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrStreamInterrupted):
		return sharederr.EnsureAppError(err, public.ErrCodeStreamInterrupted, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrUnknown):
		return sharederr.EnsureAppError(err, public.ErrCodeUnknown, ErrTypeInferenceServer)
	case errors.Is(err, domain.ErrIncompleteGeneration):
		return sharederr.EnsureAppError(err, public.ErrCodeIncompleteGeneration, ErrTypeInferenceServer)
	default:
		return sharederr.EnsureAppError(err, public.ErrCodeGenerationFailed, ErrTypeInferenceServer)
	}
}
