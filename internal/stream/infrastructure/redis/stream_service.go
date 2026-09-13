package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
)

const (
	streamPrefix     = "stream:"
	streamRetention  = 1 * time.Hour
	streamTTL        = 30 * time.Minute
	readBlockTimeout = 5 * time.Second
)

type streamService struct {
	client *redis.Client
}

func NewStreamService(client *redis.Client) *streamService {
	return &streamService{
		client: client,
	}
}

func streamKey(jobID string) string {
	return streamPrefix + jobID
}

type streamMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func (s *streamService) Close(ctx context.Context, jobID string) error {
	return s.Publish(ctx, jobID, domain.EventCompleted, "")

}

func (s *streamService) Publish(
	ctx context.Context,
	jobID string,
	eventType domain.StreamEventType,
	data string,
) error {

	msg := streamMessage{Type: string(eventType), Data: data}

	if err := s.appendMessage(ctx, jobID, msg); err != nil {
		return err
	}

	if isTerminalEvent(eventType) {
		if err := s.client.Expire(
			ctx,
			streamKey(jobID),
			streamRetention,
		).Err(); err != nil {
			return fmt.Errorf("failed to set stream retention: %w", err)
		}
	}

	return nil
}

func isTerminalEvent(eventType domain.StreamEventType) bool {
	switch eventType {
	case "completed", "failed":
		return true
	default:
		return false
	}
}

func (s *streamService) appendMessage(
	ctx context.Context,
	jobID string,
	msg streamMessage,
) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal stream message: %w", err)
	}

	_, err = s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey(jobID),
		Values: map[string]interface{}{
			"payload": string(payload),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to append stream message: %w", err)
	}

	err = s.client.ExpireNX(ctx, streamKey(jobID), streamTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set stream retention: %w", err)
	}

	return nil
}

// Subscribe subscribes to the Redis Stream for the given job ID.
//
// It starts reading from the beginning of the stream. This is intentional:
// if the worker produced chunks before the SSE handler connected,
// those chunks are still available.
func (s *streamService) Subscribe(
	ctx context.Context,
	jobID string,
) (<-chan domain.StreamEvent, error) {
	eventChan := make(chan domain.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		stream := streamKey(jobID)

		// Start from the beginning so late subscribers don't miss
		// previously generated chunks.
		lastID := "0-0"

		for {
			streams, err := s.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{
					stream,
					lastID,
				},
				Count: 100,
				Block: readBlockTimeout,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) {
					// XREAD timed out without receiving anything.
					// Continue waiting.
					continue
				}

				// Context cancellation is expected when the client
				// disconnects or the request is terminated.
				if ctx.Err() != nil {
					return
				}

				return
			}

			for _, streamResult := range streams {
				for _, message := range streamResult.Messages {
					lastID = message.ID

					payload, ok := message.Values["payload"].(string)
					if !ok {
						continue
					}

					var streamMsg streamMessage

					if err := json.Unmarshal(
						[]byte(payload),
						&streamMsg,
					); err != nil {
						continue
					}

					event := domain.StreamEvent{
						Type: streamMsg.Type,
						Data: streamMsg.Data,
					}

					select {
					case eventChan <- event:
					case <-ctx.Done():
						return
					}

					// Terminal event.
					if streamMsg.Type == "completed" ||
						streamMsg.Type == "failed" {
						return
					}
				}
			}
		}
	}()

	return eventChan, nil
}
