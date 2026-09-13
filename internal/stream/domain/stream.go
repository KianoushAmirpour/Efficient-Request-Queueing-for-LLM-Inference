package domain

import (
	"context"
)

type StreamEventType string

const (
	EventChunk     StreamEventType = "chunk"
	EventCompleted StreamEventType = "completed"
	EventFailed    StreamEventType = "failed"
	EventRetrying  StreamEventType = "retrying"
)

type StreamEvent struct {
	Type string // "chunk", "retrying", "completed", "failed"
	Data string
}

type Publisher interface {
	Publish(ctx context.Context, jobID string, eventType StreamEventType, data string) error
	Close(ctx context.Context, jobID string) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, jobID string) (<-chan StreamEvent, error)
}
