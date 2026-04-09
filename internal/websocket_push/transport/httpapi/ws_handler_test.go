package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type fakeSessionUsecase struct{}

func (f *fakeSessionUsecase) Connect(ctx context.Context, req domain.ConnectRequest) (*domain.Session, error) {
	return &domain.Session{SessionID: "s1", UserID: 1, ConnectedAt: time.Now(), LastSeenAt: time.Now()}, nil
}
func (f *fakeSessionUsecase) Disconnect(ctx context.Context, sessionID string) error { return nil }
func (f *fakeSessionUsecase) Heartbeat(ctx context.Context, sessionID string) error  { return nil }

type fakePushUsecase struct{}

func (f *fakePushUsecase) Push(ctx context.Context, msg domain.PushMessage) (*domain.PushResult, error) {
	return &domain.PushResult{}, nil
}
func (f *fakePushUsecase) Broadcast(ctx context.Context, msg domain.PushMessage) (*domain.PushResult, error) {
	return &domain.PushResult{}, nil
}

type fakeAuth struct{}

func (f *fakeAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	if token == "bad" {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "bad token", nil)
	}
	return 99, nil
}

type fakeOfflinePuller struct{}

func (f *fakeOfflinePuller) PullOffline(ctx context.Context, userID int64, limit int) ([]domain.PushMessage, error) {
	return []domain.PushMessage{{MessageID: "m1", TargetUser: userID, Payload: []byte("x"), CreatedAt: time.Now()}}, nil
}

func TestHandler_PullOffline(t *testing.T) {
	h := NewHandler(&fakeSessionUsecase{}, &fakePushUsecase{}, &fakeAuth{})
	h.SetOfflinePuller(&fakeOfflinePuller{})
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"token": "ok",
		"limit": 10,
	})
	resp, err := http.Post(ts.URL+"/ws/offline/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post pull offline: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expect status 200, got %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out["code"] != "SUCCESS" {
		t.Fatalf("expect SUCCESS code, got %v", out["code"])
	}
}
