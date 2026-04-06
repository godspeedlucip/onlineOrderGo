package domain

import (
	"context"
	"time"
)

// RepositoryPort is intentionally generic in the baseline module.
type RepositoryPort interface {
	Ping(ctx context.Context) error
}

type CachePort interface {
	Ping(ctx context.Context) error
}

type MQPort interface {
	Ping(ctx context.Context) error
}

type WebSocketPort interface {
	Ping(ctx context.Context) error
}

type PaymentPort interface {
	Ping(ctx context.Context) error
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Logger interface {
	Info(ctx context.Context, msg string, fields map[string]any)
	Error(ctx context.Context, msg string, err error, fields map[string]any)
}

type ConfigLoader interface {
	Load(ctx context.Context) (*Config, error)
}

// IdempotencyStore defines explicit state-machine transitions.
// Acquire only enters PROCESSING state when allowed.
type IdempotencyStore interface {
	Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
	Get(ctx context.Context, scene, key string) (*IdempotencyRecord, error)
	MarkSuccess(ctx context.Context, scene, key, token string, payload []byte) (updated bool, err error)
	MarkFailed(ctx context.Context, scene, key, token, reason string) (updated bool, err error)
}