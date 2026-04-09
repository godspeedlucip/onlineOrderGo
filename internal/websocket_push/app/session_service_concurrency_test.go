package app

import (
	"context"
	"sync"
	"testing"

	"go-baseline-skeleton/internal/websocket_push/domain"
	"go-baseline-skeleton/internal/websocket_push/infra/registry"
)

type staticAuth struct{}

func (a *staticAuth) ValidateToken(ctx context.Context, token string) (int64, error) { return 1, nil }

type noopGateway struct{}

func (g *noopGateway) Send(ctx context.Context, sessionID string, payload []byte) error { return nil }
func (g *noopGateway) Close(ctx context.Context, sessionID string) error                 { return nil }

func TestService_ConcurrentConnectDisconnect(t *testing.T) {
	svc := NewService(Deps{
		Registry: registry.NewMemoryRegistry(),
		Gateway:  &noopGateway{},
		Auth:     &staticAuth{},
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, err := svc.Connect(context.Background(), domain.ConnectRequest{
				Token:      "ok",
				ClientType: domain.ClientTypeUser,
			})
			if err != nil {
				t.Errorf("connect error: %v", err)
				return
			}
			_ = svc.Heartbeat(context.Background(), session.SessionID)
			_ = svc.Disconnect(context.Background(), session.SessionID)
		}()
	}
	wg.Wait()

	all, err := svc.deps.Registry.FindAll(context.Background())
	if err != nil {
		t.Fatalf("find all sessions: %v", err)
	}
	if len(all) > 1 {
		t.Fatalf("expect <=1 session after concurrent connect/disconnect, got %d", len(all))
	}
}
