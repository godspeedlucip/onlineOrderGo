package app

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type fakeConsumer struct {
	messages []domain.ConsumeMessage
}

func (c *fakeConsumer) Start(ctx context.Context, handler domain.MessageHandler) error {
	for _, m := range c.messages {
		if err := handler.HandleMessage(ctx, m); err != nil {
			continue
		}
	}
	return nil
}

type fakeCodec struct{}

func (c *fakeCodec) Encode(evt domain.OrderEvent) ([]byte, map[string]string, error) {
	b, err := json.Marshal(evt)
	return b, map[string]string{"eventType": string(evt.EventType)}, err
}

func (c *fakeCodec) Decode(msg domain.ConsumeMessage) (*domain.OrderEvent, error) {
	var evt domain.OrderEvent
	if err := json.Unmarshal(msg.Body, &evt); err != nil {
		return nil, err
	}
	if evt.EventID == "" {
		evt.EventID = msg.MessageID
	}
	return &evt, nil
}

type fakeIdem struct {
	mu    sync.Mutex
	done  map[string]bool
}

func newFakeIdem() *fakeIdem { return &fakeIdem{done: map[string]bool{}} }

func (s *fakeIdem) Acquire(ctx context.Context, eventID string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	_ = ttl
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done[eventID] {
		return "", false, nil
	}
	return eventID + "-t", true, nil
}

func (s *fakeIdem) MarkDone(ctx context.Context, eventID, token string) error {
	_ = ctx
	_ = token
	s.mu.Lock()
	s.done[eventID] = true
	s.mu.Unlock()
	return nil
}

func (s *fakeIdem) MarkFailed(ctx context.Context, eventID, token, reason string) error {
	_ = ctx
	_ = eventID
	_ = token
	_ = reason
	return nil
}

type fakeDispatcher struct {
	mu     sync.Mutex
	calls  []string
	failOn map[string]bool
}

func (d *fakeDispatcher) OnOrderCreated(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	d.mu.Lock()
	d.calls = append(d.calls, "CREATED:"+evt.EventID)
	fail := d.failOn["CREATED"]
	d.mu.Unlock()
	if fail {
		return errors.New("created handler fail")
	}
	return nil
}

func (d *fakeDispatcher) OnOrderCanceled(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	d.mu.Lock()
	d.calls = append(d.calls, "CANCELED:"+evt.EventID)
	fail := d.failOn["CANCELED"]
	d.mu.Unlock()
	if fail {
		return errors.New("canceled handler fail")
	}
	return nil
}

func (d *fakeDispatcher) OnOrderStatusChanged(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	d.mu.Lock()
	d.calls = append(d.calls, "STATUS:"+evt.EventID)
	fail := d.failOn["STATUS"]
	d.mu.Unlock()
	if fail {
		return errors.New("status handler fail")
	}
	return nil
}

type fakeOutbox struct {
	events       []domain.OrderEvent
	publishMarks int
	failedMarks  int
}

func (o *fakeOutbox) Save(ctx context.Context, evt domain.OrderEvent) error { _ = ctx; o.events = append(o.events, evt); return nil }
func (o *fakeOutbox) FetchPending(ctx context.Context, limit int) ([]domain.OrderEvent, error) {
	_ = ctx
	if limit <= 0 || len(o.events) <= limit {
		return append([]domain.OrderEvent(nil), o.events...), nil
	}
	return append([]domain.OrderEvent(nil), o.events[:limit]...), nil
}
func (o *fakeOutbox) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	_ = ctx
	_ = eventID
	_ = publishedAt
	o.publishMarks++
	return nil
}
func (o *fakeOutbox) MarkFailed(ctx context.Context, eventID, reason string, nextRetryAt time.Time) error {
	_ = ctx
	_ = eventID
	_ = reason
	_ = nextRetryAt
	o.failedMarks++
	return nil
}

type fakePublisher struct {
	mu        sync.Mutex
	failTimes int
	called    int
}

func (p *fakePublisher) Publish(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	_ = evt
	p.mu.Lock()
	defer p.mu.Unlock()
	p.called++
	if p.called <= p.failTimes {
		return errors.New("publish failed")
	}
	return nil
}

type fakeTx struct{}

func (t *fakeTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

func TestStartConsume_DuplicateMessageOnlyHandledOnce(t *testing.T) {
	evt := domain.OrderEvent{EventID: "E1", EventType: domain.EventOrderCreated, OrderID: 1}
	body, _ := json.Marshal(evt)
	dispatcher := &fakeDispatcher{failOn: map[string]bool{}}
	svc := NewService(Deps{
		Consumer:    &fakeConsumer{messages: []domain.ConsumeMessage{{MessageID: "E1", Body: body}, {MessageID: "E1", Body: body}}},
		Codec:       &fakeCodec{},
		Idempotency: newFakeIdem(),
		Dispatcher:  dispatcher,
	})
	if err := svc.StartConsume(context.Background()); err != nil {
		t.Fatalf("StartConsume failed: %v", err)
	}
	if len(dispatcher.calls) != 1 || dispatcher.calls[0] != "CREATED:E1" {
		t.Fatalf("duplicate should be skipped, calls=%v", dispatcher.calls)
	}
}

func TestStartConsume_OutOfOrderMessageProcessAsReceived(t *testing.T) {
	evt1 := domain.OrderEvent{EventID: "E2", EventType: domain.EventOrderCanceled, OrderID: 2}
	evt2 := domain.OrderEvent{EventID: "E3", EventType: domain.EventOrderCreated, OrderID: 2}
	b1, _ := json.Marshal(evt1)
	b2, _ := json.Marshal(evt2)
	dispatcher := &fakeDispatcher{failOn: map[string]bool{}}
	svc := NewService(Deps{
		Consumer:    &fakeConsumer{messages: []domain.ConsumeMessage{{MessageID: "E2", Body: b1}, {MessageID: "E3", Body: b2}}},
		Codec:       &fakeCodec{},
		Idempotency: newFakeIdem(),
		Dispatcher:  dispatcher,
	})
	if err := svc.StartConsume(context.Background()); err != nil {
		t.Fatalf("StartConsume failed: %v", err)
	}
	if len(dispatcher.calls) != 2 || dispatcher.calls[0] != "CANCELED:E2" || dispatcher.calls[1] != "CREATED:E3" {
		t.Fatalf("unexpected order calls=%v", dispatcher.calls)
	}
}

func TestFlushOutbox_PublishFailureMarkFailedThenRecover(t *testing.T) {
	outbox := &fakeOutbox{events: []domain.OrderEvent{{EventID: "E4", EventType: domain.EventOrderStatusSet, OrderID: 4}}}
	pub := &fakePublisher{failTimes: 1}
	svc := NewService(Deps{Outbox: outbox, Publisher: pub, Tx: &fakeTx{}})

	if err := svc.FlushOutbox(context.Background()); err != nil {
		t.Fatalf("first FlushOutbox failed: %v", err)
	}
	if outbox.failedMarks != 1 {
		t.Fatalf("expected mark failed once, got=%d", outbox.failedMarks)
	}

	if err := svc.FlushOutbox(context.Background()); err != nil {
		t.Fatalf("second FlushOutbox failed: %v", err)
	}
	if outbox.publishMarks != 1 {
		t.Fatalf("expected mark published once, got=%d", outbox.publishMarks)
	}
}

func TestStartConsume_HandlerErrorShouldNotStopFollowingMessages(t *testing.T) {
	evt1 := domain.OrderEvent{EventID: "E5", EventType: domain.EventOrderCreated, OrderID: 5}
	evt2 := domain.OrderEvent{EventID: "E6", EventType: domain.EventOrderCanceled, OrderID: 6}
	b1, _ := json.Marshal(evt1)
	b2, _ := json.Marshal(evt2)
	dispatcher := &fakeDispatcher{failOn: map[string]bool{"CREATED": true}}
	svc := NewService(Deps{
		Consumer:    &fakeConsumer{messages: []domain.ConsumeMessage{{MessageID: "E5", Body: b1}, {MessageID: "E6", Body: b2}}},
		Codec:       &fakeCodec{},
		Idempotency: newFakeIdem(),
		Dispatcher:  dispatcher,
	})
	if err := svc.StartConsume(context.Background()); err != nil {
		t.Fatalf("StartConsume failed: %v", err)
	}
	if len(dispatcher.calls) != 2 || dispatcher.calls[0] != "CREATED:E5" || dispatcher.calls[1] != "CANCELED:E6" {
		t.Fatalf("consumer should continue after one handler error, calls=%v", dispatcher.calls)
	}
}
