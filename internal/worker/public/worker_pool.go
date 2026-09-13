package public

import "context"

type WorkerPool interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
