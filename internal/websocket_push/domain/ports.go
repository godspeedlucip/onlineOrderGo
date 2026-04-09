package domain

import (
	"context"
	"time"
)

type PushUsecase interface {
	Push(ctx context.Context, msg PushMessage) (*PushResult, error)
	Broadcast(ctx context.Context, msg PushMessage) (*PushResult, error)
}

type SessionUsecase interface {
	Connect(ctx context.Context, req ConnectRequest) (*Session, error)
	Disconnect(ctx context.Context, sessionID string) error
	Heartbeat(ctx context.Context, sessionID string) error
}

type SessionRegistry interface {
	Add(ctx context.Context, s Session) error
	Remove(ctx context.Context, sessionID string) error
	Touch(ctx context.Context, sessionID string) error
	GetByID(ctx context.Context, sessionID string) (*Session, error)
	FindByUser(ctx context.Context, userID int64) ([]Session, error)
	FindByShop(ctx context.Context, shopID int64) ([]Session, error)
	FindByChannel(ctx context.Context, channel string) ([]Session, error)
	FindAll(ctx context.Context) ([]Session, error)
}

type PushGateway interface {
	Send(ctx context.Context, sessionID string, payload []byte) error
	Close(ctx context.Context, sessionID string) error
}

type AuthPort interface {
	ValidateToken(ctx context.Context, token string) (userID int64, err error)
}

type MQBroadcaster interface {
	PublishBroadcast(ctx context.Context, msg PushMessage) error
}

type PushDedupStore interface {
	TryAcquire(ctx context.Context, messageID string, ttl time.Duration) (bool, error)
}

type OfflineMessageStore interface {
	Save(ctx context.Context, msg PushMessage) error
	PullByUser(ctx context.Context, userID int64, limit int) ([]PushMessage, error)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Optional cross-module dependency ports for unified DI style.
type RepositoryPort interface{ Ping(ctx context.Context) error }
type CachePort interface{ Ping(ctx context.Context) error }
type MQPort interface{ Ping(ctx context.Context) error }
type WebSocketPort interface{ Ping(ctx context.Context) error }
type PaymentPort interface{ Ping(ctx context.Context) error }
