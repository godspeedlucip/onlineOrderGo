package domain

import (
	"context"
	"time"
)

type CompensationUsecase interface {
	RunOnce(ctx context.Context, job JobType) (*RunSummary, error)
	ReplayFailed(ctx context.Context, limit int) (*RunSummary, error)
}

type JobScanner interface {
	Scan(ctx context.Context, job JobType, limit int) ([]TaskItem, error)
	ScanFailed(ctx context.Context, limit int) ([]TaskItem, error)
}

type JobExecutor interface {
	Execute(ctx context.Context, item TaskItem) error
}

type TaskRepository interface {
	SaveRun(ctx context.Context, rec TaskRunRecord) error
	MarkDone(ctx context.Context, taskID string) error
	MarkFailed(ctx context.Context, taskID, reason string) error
}

type Locker interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (unlock func() error, locked bool, err error)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type MetricsRecorder interface {
	Observe(ctx context.Context, summary RunSummary)
}

// Optional dependency ports for unified DI style.
type RepositoryPort interface{ Ping(ctx context.Context) error }
type CachePort interface{ Ping(ctx context.Context) error }
type MQPort interface{ Ping(ctx context.Context) error }
type WebSocketPort interface{ Ping(ctx context.Context) error }
type PaymentPort interface{ Ping(ctx context.Context) error }
