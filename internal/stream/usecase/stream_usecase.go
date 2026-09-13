package usecase

import (
	"context"
	"log/slog"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
	"efficient-request-queueing-for-llm-inference/internal/stream/public"
)

const ErrTypeStream = "STREAM_FAILED"

type StreamPublisherUseCase struct {
	publisher domain.Publisher
}

func NewStreamPublisherUseCase(publisher domain.Publisher) *StreamPublisherUseCase {
	return &StreamPublisherUseCase{publisher: publisher}
}

func (u *StreamPublisherUseCase) Publish(ctx context.Context, jobID string, eventType public.StreamEventType, data string) error {
	var event domain.StreamEventType
	switch eventType {
	case public.EventChunk:
		event = domain.EventChunk
	case public.EventCompleted:
		event = domain.EventCompleted
	case public.EventFailed:
		event = domain.EventFailed
	case public.EventRetrying:
		event = domain.EventRetrying
	}

	err := u.publisher.Publish(ctx, jobID, event, data)
	return sharederr.EnsureAppError(err, public.ErrCodePublishFailed, ErrTypeStream)
}

func (u *StreamPublisherUseCase) Close(ctx context.Context, jobID string) error {
	err := u.publisher.Close(ctx, jobID)
	return sharederr.EnsureAppError(err, public.ErrCodeConnectionLost, ErrTypeStream)
}

type StreamSubscriberUseCase struct {
	subscriber domain.Subscriber
	logger     *slog.Logger
}

func NewStreamSubscriberUseCase(subscriber domain.Subscriber, logger *slog.Logger) *StreamSubscriberUseCase {
	return &StreamSubscriberUseCase{subscriber: subscriber, logger: logger}
}

func (u *StreamSubscriberUseCase) Subscribe(ctx context.Context, jobID string) (<-chan string, error) {
	eventChan, err := u.subscriber.Subscribe(ctx, jobID)
	if err != nil {
		return nil, sharederr.EnsureAppError(err, public.ErrCodeSubscribeFailed, ErrTypeStream)
	}

	dataChan := make(chan string, 100)
	go func() {
		defer close(dataChan)
		for event := range eventChan {
			switch event.Type {
			case "chunk":
				select {
				case dataChan <- event.Data:
				case <-ctx.Done():
					return
				}
			default:
				u.logger.DebugContext(ctx,
					"unexpected stream termination.")
				return
			}
		}
	}()

	return dataChan, nil
}
