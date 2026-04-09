package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

type fakeTx struct{}

func (f *fakeTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fakeCart struct {
	items []domain.OrderItem
	total int64
	err   error
}

func (f *fakeCart) LoadCheckedItems(ctx context.Context, userID int64) ([]domain.OrderItem, int64, error) {
	_ = ctx
	_ = userID
	if f.err != nil {
		return nil, 0, f.err
	}
	return append([]domain.OrderItem(nil), f.items...), f.total, nil
}

type fakeRepo struct {
	mu       sync.Mutex
	nextID   int64
	nextNo   int64
	saveCnt  int
	order    *domain.Order
	updateWG *sync.WaitGroup
}

func (r *fakeRepo) NextOrderNo(ctx context.Context) (string, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextNo++
	return "O_TEST_" + time.Now().Format("150405") + "_" + string(rune('A'+r.nextNo%26)), nil
}

func (r *fakeRepo) SaveOrder(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	_ = ctx
	_ = items
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	order.OrderID = r.nextID
	cp := *order
	r.order = &cp
	r.saveCnt++
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.order == nil || r.order.OrderID != orderID {
		return nil, nil
	}
	cp := *r.order
	return &cp, nil
}

func (r *fakeRepo) UpdateWithVersion(ctx context.Context, order *domain.Order, expectVersion int64) (bool, error) {
	_ = ctx
	if r.updateWG != nil {
		r.updateWG.Done()
		r.updateWG.Wait()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.order == nil || r.order.OrderID != order.OrderID {
		return false, nil
	}
	if r.order.Version != expectVersion {
		return false, nil
	}
	cp := *order
	r.order = &cp
	return true, nil
}

type fakePayment struct{ err error }

func (f *fakePayment) PreparePayment(ctx context.Context, req domain.PaymentRequest) (*domain.PaymentResponse, error) {
	_ = ctx
	_ = req
	if f.err != nil {
		return nil, f.err
	}
	return &domain.PaymentResponse{PrepayToken: "ok"}, nil
}

type fakeMQ struct {
	mu    sync.Mutex
	count int
	events []domain.OrderEvent
}

func (m *fakeMQ) PublishOrderEvent(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	m.mu.Lock()
	m.count++
	m.events = append(m.events, evt)
	m.mu.Unlock()
	return nil
}

type fakeIdem struct {
	mu      sync.Mutex
	tokens  map[string]string
	results map[string][]byte
}

func newFakeIdem() *fakeIdem {
	return &fakeIdem{tokens: map[string]string{}, results: map[string][]byte{}}
}

func (s *fakeIdem) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	_ = ttl
	k := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	if token, ok := s.tokens[k]; ok {
		return token, false, nil
	}
	token := time.Now().Format(time.RFC3339Nano)
	s.tokens[k] = token
	return token, true, nil
}

func (s *fakeIdem) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
	_ = ctx
	k := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens[k] != token {
		return nil
	}
	s.results[k] = append([]byte(nil), result...)
	return nil
}

func (s *fakeIdem) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	_ = reason
	k := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens[k] == token {
		delete(s.tokens, k)
		delete(s.results, k)
	}
	return nil
}

func (s *fakeIdem) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	_ = ctx
	k := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.results[k]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), v...), true, nil
}

func TestCreateOrder_DuplicateReplayReturnsSameSnapshot(t *testing.T) {
	repo := &fakeRepo{}
	mq := &fakeMQ{}
	svc := NewService(Deps{
		Repo:        repo,
		Cart:        &fakeCart{items: []domain.OrderItem{{ItemType: "dish", SkuID: 1, Quantity: 1, UnitAmount: 1000, LineAmount: 1000}}, total: 1000},
		Tx:          &fakeTx{},
		MQ:          mq,
		Idempotency: newFakeIdem(),
		Payment:     &fakePayment{err: errors.New("prepay down")},
	})
	cmd := domain.CreateOrderCommand{UserID: 1, AddressID: 10, IdempotencyKey: "idem-1"}
	first, err := svc.CreateOrder(context.Background(), cmd)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := svc.CreateOrder(context.Background(), cmd)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if first.OrderID != second.OrderID || first.OrderNo != second.OrderNo {
		t.Fatalf("expected replay same snapshot, first=%+v second=%+v", first, second)
	}
	if repo.saveCnt != 1 || mq.count != 1 {
		t.Fatalf("expected single persistence and event, save=%d event=%d", repo.saveCnt, mq.count)
	}
}

func TestCancelAndPayCallbackRace_OneWinsByVersion(t *testing.T) {
	repo := &fakeRepo{order: &domain.Order{OrderID: 1, OrderNo: "O1", UserID: 1, Status: domain.OrderStatusPendingPay, TotalAmount: 1000, Version: 1}}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	repo.updateWG = wg
	svc := NewService(Deps{Repo: repo, Tx: &fakeTx{}, Idempotency: newFakeIdem(), MQ: &fakeMQ{}})

	var cancelErr, payErr error
	done := sync.WaitGroup{}
	done.Add(2)
	go func() {
		defer done.Done()
		_, cancelErr = svc.CancelOrder(context.Background(), domain.CancelOrderCommand{OrderID: 1, IdempotencyKey: "c1"})
	}()
	go func() {
		defer done.Done()
		_, payErr = svc.TransitStatus(context.Background(), domain.TransitStatusCommand{OrderID: 1, To: domain.OrderStatusPaid, IdempotencyKey: "p1"})
	}()
	done.Wait()

	success := 0
	conflict := 0
	for _, err := range []error{cancelErr, payErr} {
		if err == nil {
			success++
			continue
		}
		if biz, ok := err.(*domain.BizError); ok && biz.Code == domain.CodeConflict {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("expected one success one conflict, success=%d conflict=%d cancelErr=%v payErr=%v", success, conflict, cancelErr, payErr)
	}
}

func TestTransitStatus_ConflictOnUnexpectedFrom(t *testing.T) {
	repo := &fakeRepo{order: &domain.Order{OrderID: 9, OrderNo: "O9", UserID: 1, Status: domain.OrderStatusPaid, TotalAmount: 1000, Version: 1}}
	svc := NewService(Deps{Repo: repo, Tx: &fakeTx{}, Idempotency: newFakeIdem()})
	_, err := svc.TransitStatus(context.Background(), domain.TransitStatusCommand{OrderID: 9, From: domain.OrderStatusPendingPay, To: domain.OrderStatusAccepted, IdempotencyKey: "t1"})
	if err == nil {
		t.Fatal("expected conflict")
	}
	biz, ok := err.(*domain.BizError)
	if !ok || biz.Code != domain.CodeConflict {
		t.Fatalf("expected conflict biz error, got=%v", err)
	}
}

func TestCancelPaidOrder_EmitsCancelEventForCompensationBoundary(t *testing.T) {
	repo := &fakeRepo{order: &domain.Order{OrderID: 7, OrderNo: "O7", UserID: 1, Status: domain.OrderStatusPaid, TotalAmount: 2000, Version: 1}}
	mq := &fakeMQ{}
	svc := NewService(Deps{Repo: repo, Tx: &fakeTx{}, MQ: mq, Idempotency: newFakeIdem()})
	_, err := svc.CancelOrder(context.Background(), domain.CancelOrderCommand{OrderID: 7, IdempotencyKey: "x1"})
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if mq.count != 1 || len(mq.events) != 1 || mq.events[0].Type != domain.OrderEventCanceled {
		t.Fatalf("expected one canceled event for refund/stock/coupon compensation, count=%d events=%+v", mq.count, mq.events)
	}
}
