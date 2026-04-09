package app

import (
	"context"

	"go-baseline-skeleton/internal/order_event/domain"
)

func (s *Service) StartConsume(ctx context.Context) error {
	if s.deps.Consumer == nil || s.deps.Codec == nil {
		return domain.NewBizError(domain.CodeInternal, "consume deps not initialized", nil)
	}
	return s.deps.Consumer.Start(ctx, messageHandlerFunc(func(handleCtx context.Context, msg domain.ConsumeMessage) error {
		evt, err := s.deps.Codec.Decode(msg)
		if err != nil {
			return err
		}
		if evt == nil {
			return domain.NewBizError(domain.CodeInvalidArgument, "empty event", nil)
		}
		return s.withConsumeIdempotency(handleCtx, evt.EventID, func(runCtx context.Context) error {
			return s.Handle(runCtx, *evt)
		})
	}))
}

func (s *Service) Handle(ctx context.Context, evt domain.OrderEvent) error {
	if evt.EventID == "" || evt.EventType == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid event", nil)
	}

	if s.deps.Dispatcher != nil {
		switch evt.EventType {
		case domain.EventOrderCreated:
			return s.deps.Dispatcher.OnOrderCreated(ctx, evt)
		case domain.EventOrderCanceled:
			return s.deps.Dispatcher.OnOrderCanceled(ctx, evt)
		case domain.EventOrderStatusSet:
			return s.deps.Dispatcher.OnOrderStatusChanged(ctx, evt)
		default:
			return nil
		}
	}

	switch evt.EventType {
	case domain.EventOrderCreated, domain.EventOrderCanceled, domain.EventOrderStatusSet:
		if s.deps.Repository != nil {
			_ = s.deps.Repository.Ping(ctx)
		}
		if s.deps.Cache != nil {
			_ = s.deps.Cache.Ping(ctx)
		}
		if s.deps.WebSocket != nil {
			_ = s.deps.WebSocket.Ping(ctx)
		}
		if s.deps.Payment != nil {
			_ = s.deps.Payment.Ping(ctx)
		}
		return nil
	default:
		return nil
	}
}

type messageHandlerFunc func(ctx context.Context, msg domain.ConsumeMessage) error

func (f messageHandlerFunc) HandleMessage(ctx context.Context, msg domain.ConsumeMessage) error {
	return f(ctx, msg)
}
