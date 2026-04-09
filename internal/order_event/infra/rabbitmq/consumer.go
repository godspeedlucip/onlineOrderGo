package rabbitmq

import (
	"context"
	"fmt"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"go-baseline-skeleton/internal/order_event/domain"
)

type ConsumerConfig struct {
	URL           string
	Topology      Topology
	Codec         domain.EventCodec
	MaxRetries    int
	RetryInterval time.Duration
}

type Consumer struct {
	conn          *amqp.Connection
	ch            *amqp.Channel
	topo          Topology
	maxRetries    int
	retryInterval time.Duration
}

func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	topo := cfg.Topology.withDefaults()
	if err := Declare(context.Background(), ch, topo); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := ch.Qos(topo.PrefetchCount, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = 300 * time.Millisecond
	}
	return &Consumer{conn: conn, ch: ch, topo: topo, maxRetries: cfg.MaxRetries, retryInterval: cfg.RetryInterval}, nil
}

func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Consumer) Start(ctx context.Context, handler domain.MessageHandler) error {
	if handler == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "nil message handler", nil)
	}
	if c == nil || c.ch == nil {
		return domain.NewBizError(domain.CodeInternal, "rabbitmq consumer not initialized", nil)
	}
	deliveries, err := c.ch.Consume(c.topo.Queue, c.topo.ConsumerTag, false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return domain.NewBizError(domain.CodeInternal, "rabbitmq deliveries channel closed", nil)
			}
			if err := c.handleDelivery(ctx, handler, d); err != nil {
				continue
			}
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, handler domain.MessageHandler, d amqp.Delivery) error {
	msg := domain.ConsumeMessage{
		MessageID:  d.MessageId,
		Headers:    tableToStringMap(d.Headers),
		Body:       d.Body,
		ReceivedAt: time.Now(),
	}
	if err := handler.HandleMessage(ctx, msg); err != nil {
		retryCount := readRetryCount(d.Headers)
		if retryCount < c.maxRetries {
			if repubErr := c.republishForRetry(ctx, d, retryCount+1, err.Error()); repubErr == nil {
				_ = d.Ack(false)
				return repubErr
			}
			_ = d.Nack(false, true)
			return err
		}
		_ = d.Nack(false, false)
		return err
	}
	_ = d.Ack(false)
	return nil
}

func (c *Consumer) republishForRetry(ctx context.Context, d amqp.Delivery, nextRetry int, reason string) error {
	table := d.Headers
	if table == nil {
		table = amqp.Table{}
	}
	table["x-retry-count"] = nextRetry
	table["x-last-error"] = reason
	if c.retryInterval > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.retryInterval):
		}
	}
	return c.ch.PublishWithContext(ctx, c.topo.Exchange, c.topo.RoutingKey, true, false, amqp.Publishing{
		MessageId:    d.MessageId,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
		ContentType:  d.ContentType,
		Headers:      table,
		Body:         d.Body,
	})
}

func tableToStringMap(table amqp.Table) map[string]string {
	out := map[string]string{}
	for k, v := range table {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func readRetryCount(table amqp.Table) int {
	if table == nil {
		return 0
	}
	raw, ok := table["x-retry-count"]
	if !ok || raw == nil {
		return 0
	}
	s := fmt.Sprint(raw)
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
