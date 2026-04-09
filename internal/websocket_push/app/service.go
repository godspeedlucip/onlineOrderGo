package app

import (
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Deps struct {
	Registry domain.SessionRegistry
	Gateway  domain.PushGateway
	Auth     domain.AuthPort
	MQ       domain.MQBroadcaster
	Tx       domain.TxManager
	Dedup    domain.PushDedupStore
	Offline  domain.OfflineMessageStore

	Repository domain.RepositoryPort
	Cache      domain.CachePort
	MQPort     domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	PushDedupTTL time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.PushDedupTTL <= 0 {
		deps.PushDedupTTL = 2 * time.Minute
	}
	return &Service{deps: deps}
}
