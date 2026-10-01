package domain

import (
	"context"
	"time"
)

type MetricsRecorder interface {
	ObserveDuration(duration time.Duration)
	ObserveTimeToFirstToken(duration time.Duration)
}

type LLMClient interface {
	GenerateStream(
		ctx context.Context,
		req GenerationRequest,
		onChunk func(GenerationChunk) error,
	) error
}
