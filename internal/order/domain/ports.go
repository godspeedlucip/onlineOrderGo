package domain

import (
	"context"
	"time"
)

type OrderCommandUsecase interface {
	CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*OrderView, error)
	CancelOrder(ctx context.Context, cmd CancelOrderCommand) (*OrderView, error)
	TransitStatus(ctx context.Context, cmd TransitStatusCommand) (*OrderView, error)
}

type OrderRepository interface {
	NextOrderNo(ctx context.Context) (string, error)
	SaveOrder(ctx context.Context, order *Order, items []OrderItem) error
	GetByID(ctx context.Context, orderID int64) (*Order, error)
	UpdateWithVersion(ctx context.Context, order *Order, expectVersion int64) (bool, error)
}

type CartReader interface {
	LoadCheckedItems(ctx context.Context, userID int64) ([]OrderItem, int64, error)
}

type PaymentGateway interface {
	PreparePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

type CachePort interface {
	InvalidateOrder(ctx context.Context, orderID, userID int64) error
}

type MQPort interface {
	PublishOrderEvent(ctx context.Context, evt OrderEvent) error
}

type WebSocketPort interface {
	NotifyOrderChanged(ctx context.Context, orderID int64, status OrderStatus) error
}

type IdempotencyStore interface {
	Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
	MarkDone(ctx context.Context, scene, key, token string, result []byte) error
	MarkFailed(ctx context.Context, scene, key, token, reason string) error
	GetDoneResult(ctx context.Context, scene, key string) (result []byte, found bool, err error)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
