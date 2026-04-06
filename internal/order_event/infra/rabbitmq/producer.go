package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type broker struct {
	mu     sync.RWMutex
	queues map[string]chan domain.ConsumeMessage
}

var defaultBroker = &broker{queues: make(map[string]chan domain.ConsumeMessage)}

func (b *broker) ensureQueue(name string) chan domain.ConsumeMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.queues[name]; ok {
		return ch
	}
	ch := make(chan domain.ConsumeMessage, 1024)
	b.queues[name] = ch
	return ch
}

func (b *broker) publish(queue string, msg domain.ConsumeMessage) {
	ch := b.ensureQueue(queue)
	select {
	case ch <- msg:
	default:
		// TODO: apply backpressure/metrics when queue is full.
	}
}

func (b *broker) subscribe(queue string) <-chan domain.ConsumeMessage {
	return b.ensureQueue(queue)
}

type Producer struct {
	Exchange string
	Queue    string
}

func NewProducer(exchange string) *Producer {
	return &Producer{Exchange: exchange, Queue: "order.event"}
}

func (p *Producer) Publish(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	if evt.EventID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty eventId", nil)
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	defaultBroker.publish(p.Queue, domain.ConsumeMessage{
		MessageID:  evt.EventID,
		Headers:    map[string]string{"eventType": string(evt.EventType), "version": "1", "exchange": p.Exchange},
		Body:       body,
		ReceivedAt: time.Now(),
	})
	return nil
}
