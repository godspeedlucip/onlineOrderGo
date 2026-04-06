package domain

import (
	"context"
	"time"
)

type PaymentCallbackUsecase interface {
	HandleCallback(ctx context.Context, in CallbackInput) (*CallbackAck, error)
}

type PaymentVerifier interface {
	VerifyAndParse(ctx context.Context, headers map[string]string, body []byte) (*VerifiedCallback, error)
}

type PaymentRepository interface {
	GetOrderByNo(ctx context.Context, orderNo string) (*OrderSnapshot, error)
	UpdateOrderPaidIfPending(ctx context.Context, orderID int64, payAt time.Time, txnNo string, paidAmount int64) (bool, error)
	InsertPaymentRecord(ctx context.Context, rec PaymentRecord) error
	InsertCallbackLog(ctx context.Context, log CallbackLog) error
}

type CallbackIdempotencyStore interface {
	Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
	MarkDone(ctx context.Context, scene, key, token string) error
	MarkFailed(ctx context.Context, scene, key, token, reason string) error
}

type PaymentEventPublisher interface {
	PublishOrderPaid(ctx context.Context, evt OrderPaidEvent) error
}

type GrayPolicy interface {
	Decide(ctx context.Context, cb *VerifiedCallback) (GrayDecision, error)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
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