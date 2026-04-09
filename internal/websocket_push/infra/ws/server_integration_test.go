package ws

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type fakeConnector struct {
	disconnectCount atomic.Int32
}

func (f *fakeConnector) Connect(ctx context.Context, req domain.ConnectRequest) (*domain.Session, error) {
	return &domain.Session{
		SessionID:   "s1",
		UserID:      1,
		ConnectedAt: time.Now(),
		LastSeenAt:  time.Now(),
	}, nil
}
func (f *fakeConnector) Disconnect(ctx context.Context, sessionID string) error {
	f.disconnectCount.Add(1)
	return nil
}
func (f *fakeConnector) Heartbeat(ctx context.Context, sessionID string) error { return nil }

func TestServer_HeartbeatTimeoutDisconnect(t *testing.T) {
	connector := &fakeConnector{}
	gateway := NewGatewayWithConfig(GatewayConfig{
		PongWait:     200 * time.Millisecond,
		PingInterval: 50 * time.Millisecond,
		WriteWait:    50 * time.Millisecond,
	})
	server := httptest.NewServer(NewServer(connector, gateway))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=test"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if connector.disconnectCount.Load() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expect disconnect callback after heartbeat timeout")
}
