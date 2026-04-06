package app

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) postCommit(ctx context.Context, order *domain.Order, evt *domain.OrderEvent) {
	if order == nil {
		return
	}
	if s.deps.Cache != nil {
		// Cache invalidate is best-effort and should not rollback committed transaction.
		_ = s.deps.Cache.InvalidateOrder(ctx, order.OrderID, order.UserID)
	}
	if evt != nil && s.deps.MQ != nil {
		// TODO: replace best-effort publish with outbox to ensure eventual consistency.
		_ = s.deps.MQ.PublishOrderEvent(ctx, *evt)
	}
	if s.deps.WebSocket != nil {
		// WebSocket notify is non-transactional side effect.
		_ = s.deps.WebSocket.NotifyOrderChanged(ctx, order.OrderID, order.Status)
	}
}
