package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
)

type httpServer struct {
	srv    *http.Server
	logger *slog.Logger
}

func newHTTPServer(cfg ServerConfig, engine *gin.Engine, logger *slog.Logger) *httpServer {
	return &httpServer{
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:           engine,
			ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
			ReadTimeout:       cfg.Server.ReadTimeout,
			WriteTimeout:      cfg.Server.WriteTimeout,
			IdleTimeout:       cfg.Server.IdleTimeout,
		},
		logger: logger,
	}
}

func (h *httpServer) serve() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		if err := h.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	return errCh
}

func (h *httpServer) shutdown(ctx context.Context) error {
	h.logger.InfoContext(ctx, "server shutdown initiated")
	if err := h.srv.Shutdown(ctx); err != nil {
		h.logger.ErrorContext(
			ctx,
			"server shutdown failed",
			"error.code", sharederr.ErrCodeInternal,
			"error.type", "HTTP_SERVER_FAILED",
			"error", err,
		)
		return err
	}
	h.logger.InfoContext(ctx, "server was shut down successfully.")

	return nil
}
