package public

import (
	"context"
)

type StreamEventType string

const (
	EventChunk     StreamEventType = "chunk"
	EventRetrying  StreamEventType = "retrying"
	EventCompleted StreamEventType = "completed"
	EventFailed    StreamEventType = "failed"
)

type StreamPublisher interface {
	Publish(ctx context.Context, jobID string, eventType StreamEventType, data string) error
	Close(ctx context.Context, jobID string) error
}
