package cache

import "context"

type NoopInvalidator struct{}

func NewNoopInvalidator() *NoopInvalidator { return &NoopInvalidator{} }

func (n *NoopInvalidator) InvalidateOrder(ctx context.Context, orderID, userID int64) error {
	_ = ctx
	_ = orderID
	_ = userID
	return nil
}
