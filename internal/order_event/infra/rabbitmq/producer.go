package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"go-baseline-skeleton/internal/order_event/domain"
)

type ProducerConfig struct {
	URL           string
	Topology      Topology
	PublishTimeout time.Duration
	MaxRetries    int
	RetryInterval time.Duration
	Codec         domain.EventCodec
}

type Producer struct {
	conn          *amqp.Connection
	ch            *amqp.Channel
	confirmCh     <-chan amqp.Confirmation
	topo          Topology
	codec         domain.EventCodec
	publishTimeout time.Duration
	maxRetries    int
	retryInterval time.Duration
}

func NewProducer(cfg ProducerConfig) (*Producer, error) {
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
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	codec := cfg.Codec
	if codec == nil {
		codec = NewJSONCodec()
	}
	if cfg.PublishTimeout <= 0 {
		cfg.PublishTimeout = 5 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = 500 * time.Millisecond
	}
	return &Producer{
		conn:          conn,
		ch:            ch,
		confirmCh:     ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
		topo:          topo,
		codec:         codec,
		publishTimeout: cfg.PublishTimeout,
		maxRetries:    cfg.MaxRetries,
		retryInterval: cfg.RetryInterval,
	}, nil
}

func (p *Producer) Close() error {
	if p == nil {
		return nil
	}
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *Producer) Publish(ctx context.Context, evt domain.OrderEvent) error {
	if p == nil || p.ch == nil {
		return domain.NewBizError(domain.CodeInternal, "rabbitmq producer not initialized", nil)
	}
	if evt.EventID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty eventId", nil)
	}
	body, headers, err := p.codec.Encode(evt)
	if err != nil {
		return err
	}
	attempts := p.maxRetries + 1
	for i := 0; i < attempts; i++ {
		if pubErr := p.publishOnce(ctx, evt, body, headers); pubErr == nil {
			return nil
		} else if i == attempts-1 {
			return pubErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.retryInterval):
		}
	}
	return nil
}

func (p *Producer) publishOnce(ctx context.Context, evt domain.OrderEvent, body []byte, headers map[string]string) error {
	table := amqp.Table{}
	for k, v := range headers {
		table[k] = v
	}
	table["eventId"] = evt.EventID

	publishCtx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	err := p.ch.PublishWithContext(publishCtx, p.topo.Exchange, p.topo.RoutingKey, true, false, amqp.Publishing{
		MessageId:    evt.EventID,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Headers:      table,
		Body:         body,
	})
	if err != nil {
		return err
	}
	select {
	case conf, ok := <-p.confirmCh:
		if !ok {
			return domain.NewBizError(domain.CodeInternal, "rabbitmq confirm channel closed", nil)
		}
		if !conf.Ack {
			return domain.NewBizError(domain.CodeInternal, "rabbitmq publish not acked", nil)
		}
		return nil
	case <-publishCtx.Done():
		return publishCtx.Err()
	}
}
