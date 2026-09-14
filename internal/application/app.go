package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	infraredis "efficient-request-queueing-for-llm-inference/infra/redis"
	sharederr "efficient-request-queueing-for-llm-inference/internal/shared/errors"
	"efficient-request-queueing-for-llm-inference/internal/shared/middleware"
)

type Application struct {
	registry    *Registry
	engine      *gin.Engine
	logger      *slog.Logger
	pgPool      *pgxpool.Pool
	redisClient *redis.Client
	ServerCfg   ServerConfig
}

func NewApplication(
	registry *Registry,
	engine *gin.Engine,
	logger *slog.Logger,
	pgPool *pgxpool.Pool,
	redisClient *redis.Client,
	serverCfg ServerConfig) *Application {
	return &Application{
		registry:    registry,
		engine:      engine,
		logger:      logger,
		pgPool:      pgPool,
		redisClient: redisClient,
		ServerCfg:   serverCfg,
	}
}

func (app *Application) Run(ctx context.Context) error {
	api := app.engine.Group("/api")
	api.Use(
		middleware.RecoveryMiddleware(app.logger),
		middleware.RequestContextMiddleware(app.logger),
	)

	err := app.registry.RegisterRoutes(ctx, api, app.logger)
	if err != nil {
		return err
	}

	if err := app.registry.Start(ctx, app.logger); err != nil {
		return err
	}

	httpSrv := newHTTPServer(app.ServerCfg, app.engine, app.logger)
	errCh := httpSrv.serve()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	var runErr error
	select {
	case <-shutdown:
		app.logger.WarnContext(ctx, "shutdown signal received")
	case err := <-errCh:
		app.logger.ErrorContext(
			ctx,
			"server start failed",
			"error.code", sharederr.ErrCodeInternal,
			"error.type", "HTTP_SERVER_FAILED",
			"error", err,
		)
		runErr = err
	case <-ctx.Done():
		app.logger.WarnContext(ctx, "context cancelled")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), app.ServerCfg.Server.ShutdownGrace)
	defer shutdownCancel()

	httpCtx, httpCancel := context.WithTimeout(shutdownCtx, app.ServerCfg.Server.HTTPDrainTimeout)
	defer httpCancel()

	var errs []error
	if runErr != nil {
		errs = append(errs, runErr)
	}

	if err := httpSrv.shutdown(httpCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			app.logger.WarnContext(
				ctx,
				"http drain budget exhausted; closing remaining connections",
				"error.code", sharederr.ErrCodeInternal,
				"error.type", "DRAIN_TIMEOUT_EXCEEDED",
				"error", err,
			)
		}
	} else {
		errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
	}

	err = app.registry.Shutdown(shutdownCtx, app.logger)
	if err != nil {
		errs = append(errs, fmt.Errorf("module shutdown: %w", err))
		app.logger.ErrorContext(
			ctx,
			"failed to shutdown modules",
			"error.code", sharederr.ErrCodeInternal,
			"error.type", "APP_SHUTDOWN_FAILED",
			"error", err,
		)
	}
	app.logger.InfoContext(ctx, "modules were shut down successfully.")

	if app.pgPool != nil {
		app.pgPool.Close()
	}

	if app.redisClient != nil {
		if err := infraredis.Shutdown(shutdownCtx, app.redisClient, app.logger); err != nil {
			errs = append(errs, fmt.Errorf("redis shutdown: %w", err))
			app.logger.ErrorContext(
				ctx,
				"failed to shut down redis",
				"error.code", sharederr.ErrCodeInternal,
				"error.type", "REDIS_SHUTDOWN_FAILED",
				"error", err,
			)
		}
	}
	app.logger.InfoContext(ctx, "shutdown completed")

	return errors.Join(errs...)
}
