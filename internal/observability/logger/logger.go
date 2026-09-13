package logging

import (
	"log/slog"
	"os"
)

type Config struct {
	Level     slog.Level
	AddSource bool
}

func New(cfg Config) *slog.Logger {
	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		},
	)

	contextHandler := NewContextHandler(handler)

	return slog.New(contextHandler)
}
