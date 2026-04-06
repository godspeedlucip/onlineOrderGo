package domain

import (
	"context"
	"time"
)

type CartUsecase interface {
	AddItem(ctx context.Context, cmd AddCartItemCmd, idemKey string) (*CartItemVO, error)
	SubItem(ctx context.Context, cmd SubCartItemCmd, idemKey string) (*CartItemVO, error)
	UpdateQuantity(ctx context.Context, cmd UpdateCartQtyCmd, idemKey string) (*CartItemVO, error)
	ListItems(ctx context.Context) ([]CartItemVO, error)
	Clear(ctx context.Context, idemKey string) error
}

type CartRepository interface {
	GetByKey(ctx context.Context, userID int64, key CartItemKey) (*CartItem, error)
	Create(ctx context.Context, item CartItem) (int64, error)
	UpdateQuantity(ctx context.Context, id int64, quantity int, expectedVersion int64) (bool, error)
	DeleteByID(ctx context.Context, id int64) (bool, error)
	ListByUser(ctx context.Context, userID int64) ([]CartItem, error)
	ClearByUser(ctx context.Context, userID int64) error
}

type ProductGateway interface {
	GetDishSnapshot(ctx context.Context, dishID int64) (*ItemSnapshot, error)
	GetSetmealSnapshot(ctx context.Context, setmealID int64) (*ItemSnapshot, error)
}

type UserContext interface {
	CurrentUserID(ctx context.Context) (int64, bool)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type IdempotencyStore interface {
	Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
	MarkDone(ctx context.Context, scene, key, token string) error
	MarkFailed(ctx context.Context, scene, key, token, reason string) error
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