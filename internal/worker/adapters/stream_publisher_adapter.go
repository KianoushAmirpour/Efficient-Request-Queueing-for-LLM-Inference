package adapters

import (
	"context"

	streamPublicAPI "efficient-request-queueing-for-llm-inference/internal/stream/public"
	"efficient-request-queueing-for-llm-inference/internal/worker/domain"
)

type StreamPublisherAdapter struct {
	publisher streamPublicAPI.StreamPublisher
}

var _ domain.StreamPublisher = StreamPublisherAdapter{}

func NewStreamPublisherAdapter(publisher streamPublicAPI.StreamPublisher) StreamPublisherAdapter {
	return StreamPublisherAdapter{
		publisher: publisher,
	}
}

func (a StreamPublisherAdapter) Publish(ctx context.Context, jobID string, chunk domain.StreamChunk) error {
	return a.publisher.Publish(ctx, jobID, streamPublicAPI.EventChunk, chunk.Text)
}

func (a StreamPublisherAdapter) PublishEvent(ctx context.Context, jobID string, eventType string, data string) error {
	return a.publisher.Publish(ctx, jobID, streamPublicAPI.StreamEventType(eventType), data)
}

func (a StreamPublisherAdapter) Close(ctx context.Context, jobID string) error {
	return a.publisher.Close(ctx, jobID)
}
