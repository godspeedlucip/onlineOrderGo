package websocket

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

type Notifier struct{}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) NotifyOrderChanged(ctx context.Context, orderID int64, status domain.OrderStatus) error {
	_ = ctx
	_ = orderID
	_ = status
	return nil
}
