package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) CreateOrder(ctx context.Context, cmd domain.CreateOrderCommand) (*domain.OrderView, error) {
	if s.deps.Repo == nil || s.deps.Tx == nil || s.deps.Cart == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "create order deps not initialized", nil)
	}
	if cmd.UserID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid userId", nil)
	}
	if cmd.AddressID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid addressId", nil)
	}

	return s.withIdempotency(ctx, "order:create", cmd.IdempotencyKey, func(runCtx context.Context) (*domain.OrderView, error) {
		var (
			out      *domain.OrderView
			event    *domain.OrderEvent
			orderRef *domain.Order
		)

		err := s.deps.Tx.RunInTx(runCtx, func(txCtx context.Context) error {
			items, totalAmount, err := s.deps.Cart.LoadCheckedItems(txCtx, cmd.UserID)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return domain.NewBizError(domain.CodeInvalidArgument, "cart is empty", nil)
			}
			if totalAmount <= 0 {
				return domain.NewBizError(domain.CodeInvalidArgument, "invalid order amount", nil)
			}

			orderNo, err := s.deps.Repo.NextOrderNo(txCtx)
			if err != nil {
				return err
			}
			now := time.Now()
			order := &domain.Order{
				OrderNo:     orderNo,
				UserID:      cmd.UserID,
				Status:      domain.OrderStatusPendingPay,
				TotalAmount: totalAmount,
				Remark:      cmd.Remark,
				Version:     1,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.deps.Repo.SaveOrder(txCtx, order, items); err != nil {
				return err
			}
			orderRef = order
			event = &domain.OrderEvent{
				Type:       domain.OrderEventCreated,
				OrderID:    order.OrderID,
				OrderNo:    order.OrderNo,
				From:       "",
				To:         order.Status,
				OccurredAt: now,
			}
			out = &domain.OrderView{
				OrderID:     order.OrderID,
				OrderNo:     order.OrderNo,
				Status:      order.Status,
				TotalAmount: order.TotalAmount,
				UpdatedAt:   order.UpdatedAt,
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if orderRef != nil && s.deps.Payment != nil {
			_, payErr := s.deps.Payment.PreparePayment(runCtx, domain.PaymentRequest{
				OrderID:       orderRef.OrderID,
				OrderNo:       orderRef.OrderNo,
				Amount:        orderRef.TotalAmount,
				PaymentMethod: cmd.PaymentMethod,
			})
			if payErr != nil {
				// TODO: align with Java behavior: whether payment prepare failure should fail API or remain pending.
			}
		}
		s.postCommit(runCtx, orderRef, event)
		return out, nil
	})
}
