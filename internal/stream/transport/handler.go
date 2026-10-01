package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
	"efficient-request-queueing-for-llm-inference/internal/stream/ports"
	streamPublic "efficient-request-queueing-for-llm-inference/internal/stream/public"
)

type StreamHTTPHandler struct {
	subscriber           domain.Subscriber
	AccessTokenValidator ports.AccessTokenValidator
	logger               *slog.Logger
	metrics              domain.MetricsRecorder
	jobAcceptedAtReader  domain.JobAcceptedAtReader
}

func NewStreamHTTPHandler(
	subscriber domain.Subscriber,
	accessTokenValidator ports.AccessTokenValidator,
	logger *slog.Logger,
	metrics domain.MetricsRecorder,
	jobAcceptedAtReaders domain.JobAcceptedAtReader,
) *StreamHTTPHandler {
	return &StreamHTTPHandler{
		subscriber:           subscriber,
		AccessTokenValidator: accessTokenValidator,
		logger:               logger,
		metrics:              metrics,
		jobAcceptedAtReader:  jobAcceptedAtReaders,
	}
}
func (h *StreamHTTPHandler) HandleStream(c *gin.Context) {
	jobID := c.Param("jobID")
	if jobID == "" {
		_ = c.Error(sharederr.NewAppError(
			streamPublic.ErrCodeInvalidStreamData,
			"STREAM_FAILED",
			fmt.Errorf("jobID is required"),
		))
		return
	}
	_, uuidErr := uuid.Parse(jobID)
	if uuidErr != nil {
		_ = c.Error(sharederr.NewAppError(
			streamPublic.ErrCodeInvalidStreamData,
			"STREAM_FAILED",
			fmt.Errorf("jobID is not valid"),
		))
		return
	}

	rc := http.NewResponseController(c.Writer)

	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		_ = c.Error(sharederr.NewAppError(
			sharederr.ErrCodeInternal,
			"STREAM_FAILED",
			fmt.Errorf("failed to clear write deadline"),
		))
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		_ = c.Error(sharederr.NewAppError(
			sharederr.ErrCodeInternal,
			"STREAM_FAILED",
			fmt.Errorf("streaming unsupported"),
		))
		return
	}

	ctx := c.Request.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	events, err := h.subscriber.Subscribe(ctx, jobID)
	if err != nil {
		_ = c.Error(sharederr.EnsureAppError(
			err,
			streamPublic.ErrCodeSubscribeFailed,
			"STREAM_FAILED",
		))
		return
	}

	for {
		select {
		case <-ctx.Done():
			h.logger.DebugContext(ctx, "stream ended", "jobID", jobID, "reason", "context canceled")

			return

		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": heartbeat\n\n"); err != nil {
				h.logger.DebugContext(ctx, "stream ended", "jobID", jobID, "reason", "heartbeat")
				return
			}
			flusher.Flush()

		case event, ok := <-events:
			if !ok {
				h.logger.DebugContext(ctx, "stream ended", "jobID", jobID, "reason", "no events")
				return
			}

			switch event.Type {
			case "chunk", "retrying":
				deliveryStarted := time.Now()
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}

				flusher.Flush()
				if h.metrics != nil {
					h.metrics.ObserveStreamDelivery(time.Since(deliveryStarted))
				}

			case "completed":
				createdAt := h.jobAcceptedAt(ctx, jobID)
				deliveryStarted := time.Now()
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}
				flusher.Flush()
				if h.metrics != nil {
					h.metrics.ObserveStreamDelivery(time.Since(deliveryStarted))
					h.observeJobEndToEnd(createdAt)
				}
				return

			case "failed":
				createdAt := h.jobAcceptedAt(ctx, jobID)
				deliveryStarted := time.Now()
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}

				flusher.Flush()
				if h.metrics != nil {
					h.metrics.ObserveStreamDelivery(time.Since(deliveryStarted))
					h.observeJobEndToEnd(createdAt)
				}
				return
			}
		}
	}
}

func (h *StreamHTTPHandler) jobAcceptedAt(ctx context.Context, jobID string) time.Time {
	if h.metrics == nil || h.jobAcceptedAtReader == nil {
		return time.Time{}
	}
	createdAt, err := h.jobAcceptedAtReader.GetJobAcceptedAt(ctx, jobID)
	if err != nil {
		h.logger.WarnContext(ctx, "could not read job creation time for stream metric", "jobID", jobID, "error", err)
		return time.Time{}
	}
	return createdAt
}

func (h *StreamHTTPHandler) observeJobEndToEnd(createdAt time.Time) {
	if !createdAt.IsZero() {
		h.metrics.ObserveJobEndToEnd(time.Since(createdAt))
	}
}

func writeSSEEvent(w io.Writer, eventType string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
		return err
	}

	for _, line := range strings.Split(data, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(w, "\n")
	return err
}
