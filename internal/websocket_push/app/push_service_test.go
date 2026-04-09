package app

import (
	"context"
	"testing"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type fakeRegistry struct {
	all []domain.Session
}

func (r *fakeRegistry) Add(ctx context.Context, s domain.Session) error                      { return nil }
func (r *fakeRegistry) Remove(ctx context.Context, sessionID string) error                    { return nil }
func (r *fakeRegistry) Touch(ctx context.Context, sessionID string) error                     { return nil }
func (r *fakeRegistry) GetByID(ctx context.Context, sessionID string) (*domain.Session, error) { return nil, nil }
func (r *fakeRegistry) FindByUser(ctx context.Context, userID int64) ([]domain.Session, error) { return r.all, nil }
func (r *fakeRegistry) FindByShop(ctx context.Context, shopID int64) ([]domain.Session, error) { return r.all, nil }
func (r *fakeRegistry) FindByChannel(ctx context.Context, channel string) ([]domain.Session, error) {
	return r.all, nil
}
func (r *fakeRegistry) FindAll(ctx context.Context) ([]domain.Session, error) { return r.all, nil }

type fakeGateway struct{ sends int }

func (g *fakeGateway) Send(ctx context.Context, sessionID string, payload []byte) error {
	g.sends++
	return nil
}
func (g *fakeGateway) Close(ctx context.Context, sessionID string) error { return nil }

func TestService_Push_Deduplicate(t *testing.T) {
	reg := &fakeRegistry{
		all: []domain.Session{{SessionID: "s1", UserID: 1}},
	}
	gw := &fakeGateway{}
	svc := NewService(Deps{
		Registry:     reg,
		Gateway:      gw,
		Dedup:        &memoryDedup{},
		PushDedupTTL: time.Minute,
	})
	msg := domain.PushMessage{
		MessageID:  "m1",
		TargetUser: 1,
		Payload:    []byte("hello"),
		CreatedAt:  time.Now(),
	}
	first, err := svc.Push(context.Background(), msg)
	if err != nil {
		t.Fatalf("first push: %v", err)
	}
	if first.Delivered != 1 {
		t.Fatalf("expect first delivered=1, got %d", first.Delivered)
	}
	second, err := svc.Push(context.Background(), msg)
	if err != nil {
		t.Fatalf("second push: %v", err)
	}
	if second.Delivered != 0 || second.Failed != 0 {
		t.Fatalf("expect duplicate returns zero result, got delivered=%d failed=%d", second.Delivered, second.Failed)
	}
	if gw.sends != 1 {
		t.Fatalf("expect single gateway send, got %d", gw.sends)
	}
}

type memoryDedup struct {
	m map[string]struct{}
}

func (d *memoryDedup) TryAcquire(ctx context.Context, messageID string, ttl time.Duration) (bool, error) {
	_ = ctx
	_ = ttl
	if d.m == nil {
		d.m = map[string]struct{}{}
	}
	if _, ok := d.m[messageID]; ok {
		return false, nil
	}
	d.m[messageID] = struct{}{}
	return true, nil
}
