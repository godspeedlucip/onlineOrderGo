package app

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) postCommit(ctx context.Context, order *domain.Order, evt *domain.OrderEvent) {
	_ = evt
	if order == nil {
		return
	}
	if s.deps.Cache != nil {
		_ = s.deps.Cache.InvalidateOrder(ctx, order.OrderID, order.UserID)
	}
	if s.deps.WebSocket != nil {
		_ = s.deps.WebSocket.NotifyOrderChanged(ctx, order.OrderID, order.Status)
	}
}
