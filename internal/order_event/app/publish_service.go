package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

func (s *Service) Publish(ctx context.Context, evt domain.OrderEvent) error {
	if s.deps.Tx == nil || s.deps.Outbox == nil {
		return domain.NewBizError(domain.CodeInternal, "publish deps not initialized", nil)
	}
	if evt.EventID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty eventId", nil)
	}
	if evt.EventType == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty eventType", nil)
	}
	if evt.OrderID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid orderId", nil)
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now()
	}
	if evt.Version <= 0 {
		evt.Version = 1
	}

	// Transaction boundary is controlled in app layer.
	return s.deps.Tx.RunInTx(ctx, func(txCtx context.Context) error {
		return s.deps.Outbox.Save(txCtx, evt)
	})
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	if s.deps.Outbox == nil || s.deps.Publisher == nil {
		return domain.NewBizError(domain.CodeInternal, "outbox publisher deps not initialized", nil)
	}
	pending, err := s.deps.Outbox.FetchPending(ctx, s.deps.OutboxBatchSize)
	if err != nil {
		return err
	}
	for _, evt := range pending {
		if pubErr := s.deps.Publisher.Publish(ctx, evt); pubErr != nil {
			// TODO: add retry/backoff and DLQ strategy when publish fails.
			continue
		}
		_ = s.deps.Outbox.MarkPublished(ctx, evt.EventID, time.Now())
	}
	return nil
}
