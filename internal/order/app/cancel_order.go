package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) CancelOrder(ctx context.Context, cmd domain.CancelOrderCommand) (*domain.OrderView, error) {
	if s.deps.Repo == nil || s.deps.Tx == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "cancel order deps not initialized", nil)
	}
	if cmd.OrderID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid orderId", nil)
	}

	return s.withIdempotency(ctx, "order:cancel", cmd.IdempotencyKey, func(runCtx context.Context) (*domain.OrderView, error) {
		var (
			out      *domain.OrderView
			event    *domain.OrderEvent
			orderRef *domain.Order
		)
		err := s.deps.Tx.RunInTx(runCtx, func(txCtx context.Context) error {
			order, err := s.deps.Repo.GetByID(txCtx, cmd.OrderID)
			if err != nil {
				return err
			}
			if order == nil {
				return domain.NewBizError(domain.CodeInvalidArgument, "order not found", nil)
			}
			if !domain.CanCancel(order.Status) {
				return domain.NewBizError(domain.CodeConflict, "order cannot be canceled in current status", nil)
			}

			from := order.Status
			oldVersion := order.Version
			now := time.Now()
			order.Status = domain.OrderStatusCanceled
			order.UpdatedAt = now
			order.Version++
			updated, err := s.deps.Repo.UpdateWithVersion(txCtx, order, oldVersion)
			if err != nil {
				return err
			}
			if !updated {
				return domain.NewBizError(domain.CodeConflict, "order state changed, retry", nil)
			}

			orderRef = order
			event = &domain.OrderEvent{
				Type:       domain.OrderEventCanceled,
				OrderID:    order.OrderID,
				OrderNo:    order.OrderNo,
				From:       from,
				To:         order.Status,
				OccurredAt: now,
			}
			out = &domain.OrderView{OrderID: order.OrderID, OrderNo: order.OrderNo, Status: order.Status, TotalAmount: order.TotalAmount, UpdatedAt: order.UpdatedAt}
			return nil
		})
		if err != nil {
			return nil, err
		}
		s.postCommit(runCtx, orderRef, event)
		return out, nil
	})
}
