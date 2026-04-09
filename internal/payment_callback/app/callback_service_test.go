package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type fakeVerifier struct {
	cb  *domain.VerifiedCallback
	err error
}

func (f *fakeVerifier) VerifyAndParse(ctx context.Context, headers map[string]string, body []byte) (*domain.VerifiedCallback, error) {
	_ = ctx
	_ = headers
	_ = body
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.cb
	return &cp, nil
}

type fakeRepo struct {
	mu              sync.Mutex
	order           *domain.OrderSnapshot
	updateHits      int
	paymentRecCount int
	callbackLogs    int
}

func (r *fakeRepo) GetOrderByNo(ctx context.Context, orderNo string) (*domain.OrderSnapshot, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.order == nil || r.order.OrderNo != orderNo {
		return nil, nil
	}
	cp := *r.order
	return &cp, nil
}

func (r *fakeRepo) UpdateOrderPaidIfPending(ctx context.Context, orderID int64, payAt time.Time, txnNo string, paidAmount int64) (bool, error) {
	_ = ctx
	_ = payAt
	_ = txnNo
	_ = paidAmount
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.order == nil || r.order.OrderID != orderID {
		return false, nil
	}
	if r.order.Status != 1 {
		return false, nil
	}
	r.order.Status = 2
	r.updateHits++
	return true, nil
}

func (r *fakeRepo) InsertPaymentRecord(ctx context.Context, rec domain.PaymentRecord) error {
	_ = ctx
	_ = rec
	r.mu.Lock()
	r.paymentRecCount++
	r.mu.Unlock()
	return nil
}

func (r *fakeRepo) InsertCallbackLog(ctx context.Context, log domain.CallbackLog) error {
	_ = ctx
	_ = log
	r.mu.Lock()
	r.callbackLogs++
	r.mu.Unlock()
	return nil
}

type fakeIdem struct {
	mu    sync.Mutex
	locks map[string]string
}

func newFakeIdem() *fakeIdem {
	return &fakeIdem{locks: make(map[string]string)}
}

func (s *fakeIdem) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	_ = ttl
	s.mu.Lock()
	defer s.mu.Unlock()
	k := scene + ":" + key
	if token, ok := s.locks[k]; ok {
		return token, false, nil
	}
	token := time.Now().String()
	s.locks[k] = token
	return token, true, nil
}

func (s *fakeIdem) MarkDone(ctx context.Context, scene, key, token string) error {
	_ = ctx
	_ = scene
	_ = key
	_ = token
	return nil
}

func (s *fakeIdem) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	_ = scene
	_ = key
	_ = token
	_ = reason
	return nil
}

type fakePublisher struct {
	mu    sync.Mutex
	count int
	err   error
}

func (p *fakePublisher) PublishOrderPaid(ctx context.Context, evt domain.OrderPaidEvent) error {
	_ = ctx
	_ = evt
	if p.err != nil {
		return p.err
	}
	p.mu.Lock()
	p.count++
	p.mu.Unlock()
	return nil
}

type fakeTx struct{}

func (t *fakeTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

func TestService_HandleCallback_ConcurrentDuplicateReplay(t *testing.T) {
	repo := &fakeRepo{order: &domain.OrderSnapshot{OrderID: 1, OrderNo: "ORDER_1", Status: 1, TotalAmount: 1000, MerchantID: "M001"}}
	pub := &fakePublisher{}
	svc := NewService(Deps{
		Verifier: &fakeVerifier{cb: &domain.VerifiedCallback{
			Channel:       "WECHAT",
			NotifyID:      "N1001",
			OrderNo:       "ORDER_1",
			TransactionNo: "T1001",
			PaidAmount:    1000,
			PaidAt:        time.Now(),
			RawStatus:     "SUCCESS",
			MerchantID:    "M001",
		}},
		Repo:        repo,
		Idempotency: newFakeIdem(),
		Publisher:   pub,
		Tx:          &fakeTx{},
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected callback err: %v", err)
		}
	}
	if repo.updateHits != 1 || repo.paymentRecCount != 1 || pub.count != 1 {
		t.Fatalf("duplicate callback should execute once: update=%d payment=%d publish=%d", repo.updateHits, repo.paymentRecCount, pub.count)
	}
}

func TestService_HandleCallback_NonSuccessStatusAckOnly(t *testing.T) {
	repo := &fakeRepo{order: &domain.OrderSnapshot{OrderID: 1, OrderNo: "ORDER_1", Status: 1, TotalAmount: 1000, MerchantID: "M001"}}
	svc := NewService(Deps{
		Verifier: &fakeVerifier{cb: &domain.VerifiedCallback{
			Channel:       "WECHAT",
			NotifyID:      "N2001",
			OrderNo:       "ORDER_1",
			TransactionNo: "T2001",
			PaidAmount:    1000,
			PaidAt:        time.Now(),
			RawStatus:     "NOTPAY",
			MerchantID:    "M001",
		}},
		Repo: repo,
		Tx:   &fakeTx{},
	})

	ack, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ack == nil || ack.HTTPStatus != 200 || repo.updateHits != 0 {
		t.Fatalf("expected ack-only behavior ack=%+v update=%d", ack, repo.updateHits)
	}
}

func TestService_HandleCallback_AmountMismatch(t *testing.T) {
	repo := &fakeRepo{order: &domain.OrderSnapshot{OrderID: 1, OrderNo: "ORDER_1", Status: 1, TotalAmount: 1000, MerchantID: "M001"}}
	svc := NewService(Deps{
		Verifier: &fakeVerifier{cb: &domain.VerifiedCallback{
			Channel:       "WECHAT",
			NotifyID:      "N3001",
			OrderNo:       "ORDER_1",
			TransactionNo: "T3001",
			PaidAmount:    999,
			PaidAt:        time.Now(),
			RawStatus:     "SUCCESS",
			MerchantID:    "M001",
		}},
		Repo:        repo,
		Idempotency: newFakeIdem(),
		Tx:          &fakeTx{},
	})
	_, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
	if err == nil {
		t.Fatal("expected amount mismatch error")
	}
	bizErr, ok := err.(*domain.BizError)
	if !ok || bizErr.Code != domain.CodeConflict {
		t.Fatalf("expected conflict error, got=%v", err)
	}
}

func TestService_HandleCallback_MerchantMismatch(t *testing.T) {
	repo := &fakeRepo{order: &domain.OrderSnapshot{OrderID: 1, OrderNo: "ORDER_1", Status: 1, TotalAmount: 1000, MerchantID: "M001"}}
	svc := NewService(Deps{
		Verifier: &fakeVerifier{cb: &domain.VerifiedCallback{
			Channel:       "WECHAT",
			NotifyID:      "N4001",
			OrderNo:       "ORDER_1",
			TransactionNo: "T4001",
			PaidAmount:    1000,
			PaidAt:        time.Now(),
			RawStatus:     "SUCCESS",
			MerchantID:    "M999",
		}},
		Repo:        repo,
		Idempotency: newFakeIdem(),
		Tx:          &fakeTx{},
	})
	_, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
	if err == nil {
		t.Fatal("expected merchant mismatch error")
	}
	bizErr, ok := err.(*domain.BizError)
	if !ok || bizErr.Code != domain.CodeConflict {
		t.Fatalf("expected conflict error, got=%v", err)
	}
}

func TestService_HandleCallback_SignatureFailure(t *testing.T) {
	svc := NewService(Deps{
		Verifier: &fakeVerifier{err: errors.New("bad sign")},
		Repo:     &fakeRepo{},
		Tx:       &fakeTx{},
	})
	_, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	bizErr, ok := err.(*domain.BizError)
	if !ok || bizErr.Code != domain.CodeUnauthorized {
		t.Fatalf("expected unauthorized error, got=%v", err)
	}
}

func TestService_HandleCallback_PublisherFailedShouldReturnError(t *testing.T) {
	repo := &fakeRepo{order: &domain.OrderSnapshot{OrderID: 1, OrderNo: "ORDER_1", Status: 1, TotalAmount: 1000, MerchantID: "M001"}}
	pub := &fakePublisher{err: errors.New("outbox insert failed")}
	svc := NewService(Deps{
		Verifier: &fakeVerifier{cb: &domain.VerifiedCallback{
			Channel:       "WECHAT",
			NotifyID:      "N5001",
			OrderNo:       "ORDER_1",
			TransactionNo: "T5001",
			PaidAmount:    1000,
			PaidAt:        time.Now(),
			RawStatus:     "SUCCESS",
			MerchantID:    "M001",
		}},
		Repo:        repo,
		Idempotency: newFakeIdem(),
		Publisher:   pub,
		Tx:          &fakeTx{},
	})
	_, err := svc.HandleCallback(context.Background(), domain.CallbackInput{Body: []byte(`{"x":1}`)})
	if err == nil {
		t.Fatal("expected publish failure")
	}
}
