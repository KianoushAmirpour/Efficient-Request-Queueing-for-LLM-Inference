package openai

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"efficient-request-queueing-for-llm-inference/internal/inference_server/domain"
)

type OpenaiClient struct {
	client openai.Client
}

func NewOpenAIClient(
	apiKey string,
) *OpenaiClient {

	transport := &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := openai.NewClient(
		option.WithBaseURL("https://api.avalai.ir/v1"),
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(3),
		option.WithHTTPClient(&http.Client{
			Timeout:   10 * time.Minute,
			Transport: transport,
		}),
	)

	return &OpenaiClient{
		client: client,
	}
}

func (c *OpenaiClient) GenerateStream(
	ctx context.Context,
	req domain.GenerationRequest,
	onChunk func(domain.GenerationChunk) error,
) error {

	stream := c.client.Chat.Completions.NewStreaming(
		ctx,
		openai.ChatCompletionNewParams{
			Model: openai.ChatModel(req.Model),
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(req.Prompt),
			},
			MaxTokens:   openai.Int(int64(req.MaxTokens)),
			Temperature: openai.Float(float64(req.Temperature)),
		},
	)

	defer func() {
		_ = stream.Close()
	}()

	var finishReason string

	for stream.Next() {

		event := stream.Current()

		for _, choice := range event.Choices {

			if choice.FinishReason != "" {
				finishReason = string(choice.FinishReason)
			}

			text := choice.Delta.Content

			if text == "" {
				continue
			}

			if err := onChunk(domain.GenerationChunk{Text: text}); err != nil {
				return fmt.Errorf("chunk processing failed: %w", domain.ErrStreamInterrupted)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("streaming generation: %w", classifyOpenAIError(err))
	}

	if finishReason == "" {
		// Stream ended without a clear finish reason
		return domain.ErrIncompleteGeneration
	}

	switch finishReason {
	case "length":
		return domain.ErrMaxTokensExceeded
	case "content_filter":
		return domain.ErrContentFiltered
	}

	return nil
}

func classifyOpenAIError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("%w: %s", domain.ErrInvalidRequest, apiErr.Message)
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("%w: %s", domain.ErrAuthenticationFailed, apiErr.Message)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s", domain.ErrModelNotFound, apiErr.Message)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %s", domain.ErrRateLimitExceeded, apiErr.Message)
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
			return fmt.Errorf("%w: %s", domain.ErrServerOverloaded, apiErr.Message)
		default:
			return fmt.Errorf("%w: %s (status: %d)", domain.ErrUnknown, apiErr.Message, apiErr.StatusCode)
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", domain.ErrServerOverloaded, err)
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %w", domain.ErrStreamInterrupted, err)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return fmt.Errorf("%w: %w", domain.ErrServerOverloaded, err)
		}

		return fmt.Errorf("%w: %w", domain.ErrServerOverloaded, err)
	}

	return fmt.Errorf("%w: %w", domain.ErrUnknown, err)
}
