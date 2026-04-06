package app

import (
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

type Deps struct {
	Repo        domain.OrderRepository
	Cart        domain.CartReader
	Payment     domain.PaymentGateway
	Cache       domain.CachePort
	MQ          domain.MQPort
	WebSocket   domain.WebSocketPort
	Idempotency domain.IdempotencyStore
	Tx          domain.TxManager

	IdempotencyTTL time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.IdempotencyTTL <= 0 {
		deps.IdempotencyTTL = 5 * time.Minute
	}
	return &Service{deps: deps}
}
