package domain

type GenerationRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float32
}

type GenerationChunk struct {
	Text string
}

type GenerationResponse struct {
	Text string
}
