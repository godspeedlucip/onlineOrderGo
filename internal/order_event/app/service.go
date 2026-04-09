package app

import (
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Deps struct {
	Publisher   domain.EventPublisher
	Consumer    domain.EventConsumer
	Codec       domain.EventCodec
	Idempotency domain.ConsumeIdempotencyStore
	Outbox      domain.OutboxRepository
	Tx          domain.TxManager
	Dispatcher  domain.EventDispatcher

	Repository domain.RepositoryPort
	Cache      domain.CachePort
	MQ         domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	ConsumeIdempotencyTTL time.Duration
	OutboxBatchSize       int
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.ConsumeIdempotencyTTL <= 0 {
		deps.ConsumeIdempotencyTTL = 10 * time.Minute
	}
	if deps.OutboxBatchSize <= 0 {
		deps.OutboxBatchSize = 100
	}
	return &Service{deps: deps}
}
