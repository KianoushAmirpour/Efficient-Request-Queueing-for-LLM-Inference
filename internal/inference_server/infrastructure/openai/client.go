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
	config "efficient-request-queueing-for-llm-inference/internal/inference_server/infrastructure/config"
)

type OpenaiClient struct {
	client            openai.Client
	streamIdleTimeout time.Duration
}

func NewOpenAIClient(
	cfg config.InferenceClient,
) *OpenaiClient {

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConnections,
		MaxIdleConnsPerHost: cfg.MaxIdleConnectionsHost,
		IdleConnTimeout:     cfg.ConnectionIdleTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
	}

	client := openai.NewClient(
		option.WithBaseURL(cfg.BaseURL),
		option.WithAPIKey(cfg.APIKey),
		option.WithMaxRetries(cfg.MaxRetries),
		option.WithHTTPClient(&http.Client{
			Timeout:   cfg.StreamTotalTimeout,
			Transport: transport,
		}),
	)

	return &OpenaiClient{
		client:            client,
		streamIdleTimeout: cfg.StreamIdleTimeout,
	}
}

func (c *OpenaiClient) GenerateStream(
	ctx context.Context,
	req domain.GenerationRequest,
	onChunk func(domain.GenerationChunk) error,
) error {

	streamCtx, cancelStream := context.WithCancelCause(ctx)
	defer cancelStream(nil)

	idleTimer := time.AfterFunc(c.streamIdleTimeout, func() {
		cancelStream(domain.ErrStreamIdleTimeout)
	})
	defer idleTimer.Stop()

	stream := c.client.Chat.Completions.NewStreaming(
		streamCtx,
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
		idleTimer.Stop()

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

		idleTimer.Reset(c.streamIdleTimeout)
	}

	if errors.Is(context.Cause(streamCtx), domain.ErrStreamIdleTimeout) {
		return fmt.Errorf("streaming generation: %w: %w", domain.ErrServerOverloaded, domain.ErrStreamIdleTimeout)
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
