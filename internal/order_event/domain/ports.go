package domain

import (
	"context"
	"time"
)

type PublishUsecase interface {
	Publish(ctx context.Context, evt OrderEvent) error
}

type ConsumeUsecase interface {
	Handle(ctx context.Context, evt OrderEvent) error
}

type EventPublisher interface {
	Publish(ctx context.Context, evt OrderEvent) error
}

type EventConsumer interface {
	Start(ctx context.Context, handler MessageHandler) error
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, msg ConsumeMessage) error
}

type EventCodec interface {
	Encode(evt OrderEvent) ([]byte, map[string]string, error)
	Decode(msg ConsumeMessage) (*OrderEvent, error)
}

type ConsumeIdempotencyStore interface {
	Acquire(ctx context.Context, eventID string, ttl time.Duration) (token string, acquired bool, err error)
	MarkDone(ctx context.Context, eventID, token string) error
	MarkFailed(ctx context.Context, eventID, token, reason string) error
}

type OutboxRepository interface {
	Save(ctx context.Context, evt OrderEvent) error
	FetchPending(ctx context.Context, limit int) ([]OrderEvent, error)
	MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID, reason string, nextRetryAt time.Time) error
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type EventDispatcher interface {
	OnOrderCreated(ctx context.Context, evt OrderEvent) error
	OnOrderCanceled(ctx context.Context, evt OrderEvent) error
	OnOrderStatusChanged(ctx context.Context, evt OrderEvent) error
}

// Optional cross-module ports for app-level integration.
type RepositoryPort interface{ Ping(ctx context.Context) error }
type CachePort interface{ Ping(ctx context.Context) error }
type MQPort interface{ Ping(ctx context.Context) error }
type WebSocketPort interface{ Ping(ctx context.Context) error }
type PaymentPort interface{ Ping(ctx context.Context) error }
