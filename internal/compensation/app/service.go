package app

import (
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type Deps struct {
	Scanner  domain.JobScanner
	Executor domain.JobExecutor
	Repo     domain.TaskRepository
	Lock     domain.Locker
	Tx       domain.TxManager

	Repository domain.RepositoryPort
	Cache      domain.CachePort
	MQ         domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	BatchSize int
	LockTTL   time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.BatchSize <= 0 {
		deps.BatchSize = 200
	}
	if deps.LockTTL <= 0 {
		deps.LockTTL = 30 * time.Second
	}
	return &Service{deps: deps}
}
