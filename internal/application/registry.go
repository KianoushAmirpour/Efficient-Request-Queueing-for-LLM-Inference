package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Registry struct {
	modules []Module
}

func (r *Registry) Register(m ...Module) {
	r.modules = append(r.modules, m...)
}

func (r *Registry) RegisterRoutes(ctx context.Context, rg *gin.RouterGroup, logger *slog.Logger) error {
	for _, m := range r.modules {
		if h, ok := m.(HttpModule); ok {
			if err := h.RegisterRoutes(rg); err != nil {
				logger.ErrorContext(ctx, "failed to register routes", "module", m.Name(), "error", err)
				return fmt.Errorf("register routes %s: %w", m.Name(), err)
			}
		}
	}
	return nil
}

func (r *Registry) Start(ctx context.Context, logger *slog.Logger) error {
	started := make([]StoppableModule, 0, len(r.modules))
	for _, m := range r.modules {
		if s, ok := m.(StartableModule); ok {
			if err := s.Start(ctx); err != nil {
				logger.ErrorContext(ctx, "failed to start the module", "module", m.Name(), "error", err)

				rollbackErrs := []error{fmt.Errorf("start %s: %w", m.Name(), err)}
				for i := len(started) - 1; i >= 0; i-- {
					startedModule := started[i]
					if stopErr := startedModule.Stop(ctx); stopErr != nil {
						logger.ErrorContext(
							ctx,
							"failed to rollback started module",
							"module", startedModule.Name(),
							"error", stopErr,
						)
						rollbackErrs = append(
							rollbackErrs,
							fmt.Errorf("rollback %s: %w", startedModule.Name(), stopErr),
						)
					}
				}
				return errors.Join(rollbackErrs...)
			}
			if s, ok := m.(StoppableModule); ok {
				started = append(started, s)
			}
		}
	}
	return nil
}

func (r *Registry) Shutdown(ctx context.Context, logger *slog.Logger) error {
	var errs []error
	for i := len(r.modules) - 1; i >= 0; i-- {
		m := r.modules[i]
		s, ok := m.(StoppableModule)
		if !ok {
			continue
		}
		if err := s.Stop(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to stop the module", "module", m.Name(), "error", err)
			errs = append(errs, fmt.Errorf("error shutting down %s: %w", m.Name(), err))
		}
	}
	return errors.Join(errs...)
}
