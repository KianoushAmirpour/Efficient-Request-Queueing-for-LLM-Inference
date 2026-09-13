package adapters

import (
	"context"

	"efficient-request-queueing-for-llm-inference/internal/recovery/domain"
	streamPublic "efficient-request-queueing-for-llm-inference/internal/stream/public"
)

type EventsAdapter struct {
	publisher streamPublic.StreamPublisher
}

var _ domain.Events = EventsAdapter{}

func NewEventsAdapter(publisher streamPublic.StreamPublisher) EventsAdapter {
	return EventsAdapter{publisher: publisher}
}

func (a EventsAdapter) PublishEvent(ctx context.Context, jobID, eventType, data string) error {
	return a.publisher.Publish(ctx, jobID, streamPublic.StreamEventType(eventType), data)
}

func (a EventsAdapter) Close(ctx context.Context, jobID string) error {
	return a.publisher.Close(ctx, jobID)
}
