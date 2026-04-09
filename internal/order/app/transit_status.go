package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) TransitStatus(ctx context.Context, cmd domain.TransitStatusCommand) (*domain.OrderView, error) {
	if s.deps.Repo == nil || s.deps.Tx == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "transit status deps not initialized", nil)
	}
	if cmd.OrderID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid orderId", nil)
	}
	if cmd.To == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty target status", nil)
	}

	return s.withIdempotency(ctx, "order:transit", cmd.IdempotencyKey, func(runCtx context.Context) (*domain.OrderView, error) {
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
			if cmd.From != "" && order.Status != cmd.From {
				return domain.NewBizError(domain.CodeConflict, "unexpected order status", nil)
			}
			if !domain.CanTransit(order.Status, cmd.To) {
				return domain.NewBizError(domain.CodeConflict, "illegal status transition", nil)
			}

			from := order.Status
			oldVersion := order.Version
			now := time.Now()
			order.Status = cmd.To
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
			event = &domain.OrderEvent{Type: domain.OrderEventStatusChanged, OrderID: order.OrderID, OrderNo: order.OrderNo, From: from, To: order.Status, OccurredAt: now}
			if s.deps.MQ != nil {
				if err := s.deps.MQ.PublishOrderEvent(txCtx, *event); err != nil {
					return err
				}
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
