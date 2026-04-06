package ws

import (
	"context"
	"sync"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Gateway struct {
	mu      sync.RWMutex
	clients map[string]struct{}
}

func NewGateway() *Gateway {
	return &Gateway{clients: make(map[string]struct{})}
}

func (g *Gateway) Register(sessionID string) {
	g.mu.Lock()
	g.clients[sessionID] = struct{}{}
	g.mu.Unlock()
}

func (g *Gateway) Send(ctx context.Context, sessionID string, payload []byte) error {
	_ = ctx
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	if len(payload) == 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty payload", nil)
	}
	g.mu.RLock()
	_, ok := g.clients[sessionID]
	g.mu.RUnlock()
	if !ok {
		return domain.NewBizError(domain.CodeConflict, "session offline", nil)
	}
	// TODO: write payload to real websocket connection.
	return nil
}

func (g *Gateway) Close(ctx context.Context, sessionID string) error {
	_ = ctx
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	g.mu.Lock()
	delete(g.clients, sessionID)
	g.mu.Unlock()
	return nil
}
