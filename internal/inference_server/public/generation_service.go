package public

import (
	"context"
)

type GenerationInput struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float32
}

type GenerationOutput struct {
	Text string
}

type GenerationChunk struct {
	Text string
}

type StreamGenerator interface {
	GenerateStream(ctx context.Context, input GenerationInput, onChunk func(GenerationChunk) error) error
}
