package rabbitmq

import (
	"context"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Topology struct {
	Exchange string
	Queue    string
	DLQ      string
}

func (t Topology) Declare(ctx context.Context) error {
	_ = ctx
	if t.Exchange == "" || t.Queue == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty exchange or queue", nil)
	}
	_ = defaultBroker.ensureQueue(t.Queue)
	if t.DLQ != "" {
		_ = defaultBroker.ensureQueue(t.DLQ)
	}
	// TODO: map to real durable exchange/queue/binding declarations in RabbitMQ.
	return nil
}
