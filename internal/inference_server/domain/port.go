package domain

import (
	"context"
)

type LLMClient interface {
	GenerateStream(
		ctx context.Context,
		req GenerationRequest,
		onChunk func(GenerationChunk) error,
	) error
}
