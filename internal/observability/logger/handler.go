package logging

import (
	"context"
	"log/slog"
)

type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(handler slog.Handler) *ContextHandler {
	return &ContextHandler{
		Handler: handler,
	}
}

func (h *ContextHandler) Handle(
	ctx context.Context,
	record slog.Record,
) error {
	if ctx == nil {
		return h.Handler.Handle(ctx, record)
	}

	if requestID := RequestID(ctx); requestID != "" {
		record.AddAttrs(
			slog.String("request.id", requestID),
		)
	}

	if userID := UserID(ctx); userID != "" {
		record.AddAttrs(
			slog.String("user.id", userID),
		)
	}

	return h.Handler.Handle(ctx, record)
}

func (h *ContextHandler) WithAttrs(
	attrs []slog.Attr,
) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *ContextHandler) WithGroup(
	name string,
) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
