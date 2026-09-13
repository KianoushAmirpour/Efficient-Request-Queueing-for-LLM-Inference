package domain

import "context"

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(ctx context.Context) (string, error)) (string, error)
}
