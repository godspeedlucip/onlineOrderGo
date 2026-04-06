package rabbitmq

import (
	"context"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Consumer struct {
	Queue string
}

func NewConsumer(queue string) *Consumer {
	if queue == "" {
		queue = "order.event"
	}
	return &Consumer{Queue: queue}
}

func (c *Consumer) Start(ctx context.Context, handler domain.MessageHandler) error {
	if handler == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "nil message handler", nil)
	}
	stream := defaultBroker.subscribe(c.Queue)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-stream:
			if err := handler.HandleMessage(ctx, msg); err != nil {
				// TODO: add nack/requeue/dead-letter strategy.
				continue
			}
		}
	}
}
