package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Topology struct {
	Exchange        string
	Queue           string
	RoutingKey      string
	ConsumerTag     string
	DLXExchange     string
	DLQ             string
	DLQRoutingKey   string
	PrefetchCount   int
}

func (t Topology) withDefaults() Topology {
	if t.Exchange == "" {
		t.Exchange = "order.event.exchange"
	}
	if t.Queue == "" {
		t.Queue = "order.event"
	}
	if t.RoutingKey == "" {
		t.RoutingKey = "order.event"
	}
	if t.DLXExchange == "" {
		t.DLXExchange = "order.event.dlx"
	}
	if t.DLQ == "" {
		t.DLQ = "order.event.dlq"
	}
	if t.DLQRoutingKey == "" {
		t.DLQRoutingKey = t.DLQ
	}
	if t.PrefetchCount <= 0 {
		t.PrefetchCount = 20
	}
	return t
}

func Declare(ctx context.Context, ch *amqp.Channel, topo Topology) error {
	_ = ctx
	if ch == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "nil rabbitmq channel", nil)
	}
	t := topo.withDefaults()

	if err := ch.ExchangeDeclare(t.Exchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(t.DLXExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    t.DLXExchange,
		"x-dead-letter-routing-key": t.DLQRoutingKey,
	}
	if _, err := ch.QueueDeclare(t.Queue, true, false, false, false, queueArgs); err != nil {
		return err
	}
	if err := ch.QueueBind(t.Queue, t.RoutingKey, t.Exchange, false, nil); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(t.DLQ, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(t.DLQ, t.DLQRoutingKey, t.DLXExchange, false, nil); err != nil {
		return err
	}
	return nil
}
