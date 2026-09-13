package transport

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/stream/domain"
	"efficient-request-queueing-for-llm-inference/internal/stream/ports"
	streamPublic "efficient-request-queueing-for-llm-inference/internal/stream/public"
)

type StreamHTTPHandler struct {
	subscriber           domain.Subscriber
	AccessTokenValidator ports.AccessTokenValidator
	logger               *slog.Logger
}

func NewStreamHTTPHandler(
	subscriber domain.Subscriber,
	accessTokenValidator ports.AccessTokenValidator,
	logger *slog.Logger,
) *StreamHTTPHandler {
	return &StreamHTTPHandler{
		subscriber:           subscriber,
		AccessTokenValidator: accessTokenValidator,
		logger:               logger,
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
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}

				flusher.Flush()

			case "completed":
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}
				flusher.Flush()
				return

			case "failed":
				if err := writeSSEEvent(c.Writer, event.Type, event.Data); err != nil {
					return
				}

				flusher.Flush()
				return
			}
		}
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
