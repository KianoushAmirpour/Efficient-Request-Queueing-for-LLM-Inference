package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type unitOfWork struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewUnitOfWork(db *pgxpool.Pool, logger *slog.Logger) *unitOfWork {
	return &unitOfWork{db: db, logger: logger}
}

func (u *unitOfWork) Execute(
	ctx context.Context,
	fn func(ctx context.Context) (string, error),
) (string, error) {

	tx, err := u.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("start transaction: %w", err)
	}

	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			u.logger.WarnContext(ctx, "transaction rollback failed", "error", rbErr)
		}
	}()

	txCtx := injectTx(ctx, tx)

	result, err := fn(txCtx)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			return "", fmt.Errorf("transaction already rolled back: %w", err)
		}
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}
