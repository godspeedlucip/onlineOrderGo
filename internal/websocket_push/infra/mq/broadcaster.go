package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Config struct {
	DSN         string
	Exchange    string
	DLQExchange string
	NodeQueue   string
	ConsumerTag string
	Prefetch    int
	MaxRetry    int
}

type Broadcaster struct {
	cfg     Config
	conn    *amqp.Connection
	channel *amqp.Channel
	confirm <-chan amqp.Confirmation
}

func NewBroadcaster(cfg Config) (*Broadcaster, error) {
	conn, err := amqp.Dial(cfg.DSN)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if cfg.Exchange == "" {
		cfg.Exchange = "ws.push.broadcast"
	}
	if cfg.DLQExchange == "" {
		cfg.DLQExchange = "ws.push.broadcast.dlq"
	}
	if cfg.NodeQueue == "" {
		cfg.NodeQueue = "ws.push.node.default"
	}
	if cfg.ConsumerTag == "" {
		cfg.ConsumerTag = "ws_push_consumer"
	}
	if cfg.Prefetch <= 0 {
		cfg.Prefetch = 50
	}
	if cfg.MaxRetry <= 0 {
		cfg.MaxRetry = 5
	}
	b := &Broadcaster{cfg: cfg, conn: conn, channel: ch}
	if err := b.initTopology(); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	b.confirm = b.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	return b, nil
}

func NewNoopBroadcaster() *Broadcaster {
	return &Broadcaster{}
}

func (b *Broadcaster) PublishBroadcast(ctx context.Context, msg domain.PushMessage) error {
	if b.channel == nil {
		return nil
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := b.channel.PublishWithContext(ctx, b.cfg.Exchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		MessageId:    msg.MessageID,
		Type:         msg.EventType,
	}); err != nil {
		return err
	}
	select {
	case confirm, ok := <-b.confirm:
		if !ok || !confirm.Ack {
			return errors.New("publish not acknowledged")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("publish confirm timeout")
	}
}

func (b *Broadcaster) Consume(ctx context.Context, handler func(context.Context, domain.PushMessage) error) error {
	if b.channel == nil {
		return nil
	}
	if err := b.channel.Qos(b.cfg.Prefetch, 0, false); err != nil {
		return err
	}
	deliveries, err := b.channel.Consume(
		b.cfg.NodeQueue,
		b.cfg.ConsumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("ws broadcast consumer closed")
			}
			var msg domain.PushMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				_ = d.Ack(false)
				continue
			}
			if err := handler(ctx, msg); err != nil {
				if b.retryCount(d.Headers) >= b.cfg.MaxRetry {
					_ = b.publishDLQ(ctx, d)
					_ = d.Ack(false)
					continue
				}
				_ = b.publishRetry(ctx, d)
				_ = d.Ack(false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

func (b *Broadcaster) Close() error {
	var err error
	if b.channel != nil {
		err = b.channel.Close()
	}
	if b.conn != nil {
		connErr := b.conn.Close()
		if err == nil {
			err = connErr
		}
	}
	return err
}

func (b *Broadcaster) initTopology() error {
	if b.channel == nil {
		return nil
	}
	if err := b.channel.Confirm(false); err != nil {
		return err
	}
	if err := b.channel.ExchangeDeclare(b.cfg.Exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if err := b.channel.ExchangeDeclare(b.cfg.DLQExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := b.channel.QueueDeclare(b.cfg.NodeQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := b.channel.QueueBind(b.cfg.NodeQueue, "", b.cfg.Exchange, false, nil); err != nil {
		return err
	}
	_, err := b.channel.QueueDeclare(b.cfg.NodeQueue+".dlq", true, false, false, false, nil)
	if err != nil {
		return err
	}
	return b.channel.QueueBind(b.cfg.NodeQueue+".dlq", "", b.cfg.DLQExchange, false, nil)
}

func (b *Broadcaster) publishRetry(ctx context.Context, d amqp.Delivery) error {
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = b.retryCount(d.Headers) + 1
	return b.channel.PublishWithContext(ctx, "", b.cfg.NodeQueue, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		Body:         d.Body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Headers:      headers,
		MessageId:    d.MessageId,
		Type:         d.Type,
	})
}

func (b *Broadcaster) publishDLQ(ctx context.Context, d amqp.Delivery) error {
	return b.channel.PublishWithContext(ctx, b.cfg.DLQExchange, "", false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		Body:         d.Body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Headers:      d.Headers,
		MessageId:    d.MessageId,
		Type:         d.Type,
	})
}

func (b *Broadcaster) retryCount(headers amqp.Table) int {
	v, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int32:
		return int(val)
	case int64:
		return int(val)
	case int:
		return val
	case string:
		var out int
		_, _ = fmt.Sscanf(val, "%d", &out)
		return out
	default:
		return 0
	}
}
