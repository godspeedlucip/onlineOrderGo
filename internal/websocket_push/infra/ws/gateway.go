package ws

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Gateway struct {
	mu      sync.RWMutex
	clients map[string]*client

	writeWait      time.Duration
	pongWait       time.Duration
	pingInterval   time.Duration
	maxMessageSize int64
}

type GatewayConfig struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingInterval   time.Duration
	MaxMessageSize int64
}

type client struct {
	sessionID    string
	conn         *websocket.Conn
	send         chan []byte
	done         chan struct{}
	onHeartbeat  func()
	onDisconnect func()
	once         sync.Once
}

func NewGateway() *Gateway {
	return NewGatewayWithConfig(GatewayConfig{})
}

func NewGatewayWithConfig(cfg GatewayConfig) *Gateway {
	pongWait := 60 * time.Second
	if cfg.PongWait > 0 {
		pongWait = cfg.PongWait
	}
	writeWait := 10 * time.Second
	if cfg.WriteWait > 0 {
		writeWait = cfg.WriteWait
	}
	pingInterval := pongWait * 9 / 10
	if cfg.PingInterval > 0 {
		pingInterval = cfg.PingInterval
	}
	maxMessageSize := int64(64 * 1024)
	if cfg.MaxMessageSize > 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	return &Gateway{
		clients:        make(map[string]*client),
		writeWait:      writeWait,
		pongWait:       pongWait,
		pingInterval:   pingInterval,
		maxMessageSize: maxMessageSize,
	}
}

func (g *Gateway) Send(ctx context.Context, sessionID string, payload []byte) error {
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	if len(payload) == 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty payload", nil)
	}
	g.mu.RLock()
	c, ok := g.clients[sessionID]
	g.mu.RUnlock()
	if !ok {
		return domain.NewBizError(domain.CodeConflict, "session offline", nil)
	}

	select {
	case c.send <- payload:
		return nil
	case <-c.done:
		return domain.NewBizError(domain.CodeConflict, "session offline", nil)
	case <-ctx.Done():
		return domain.NewBizError(domain.CodeConflict, "send canceled", ctx.Err())
	case <-time.After(g.writeWait):
		return domain.NewBizError(domain.CodeConflict, "send timeout", nil)
	}
}

func (g *Gateway) Close(ctx context.Context, sessionID string) error {
	_ = ctx
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	g.mu.Lock()
	c, ok := g.clients[sessionID]
	if ok {
		delete(g.clients, sessionID)
	}
	g.mu.Unlock()
	if ok {
		c.close()
	}
	return nil
}

func (g *Gateway) Attach(sessionID string, conn *websocket.Conn, onHeartbeat func(), onDisconnect func()) {
	c := &client{
		sessionID:    sessionID,
		conn:         conn,
		send:         make(chan []byte, 128),
		done:         make(chan struct{}),
		onHeartbeat:  onHeartbeat,
		onDisconnect: onDisconnect,
	}
	g.mu.Lock()
	if old, exists := g.clients[sessionID]; exists {
		delete(g.clients, sessionID)
		old.close()
	}
	g.clients[sessionID] = c
	g.mu.Unlock()

	go g.readPump(c)
	go g.writePump(c)
}

func (g *Gateway) readPump(c *client) {
	defer c.close()
	c.conn.SetReadLimit(g.maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(g.pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(g.pongWait))
		if c.onHeartbeat != nil {
			c.onHeartbeat()
		}
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		if c.onHeartbeat != nil {
			c.onHeartbeat()
		}
	}
}

func (g *Gateway) writePump(c *client) {
	ticker := time.NewTicker(g.pingInterval)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(g.writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-c.done:
			return
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(g.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
	})
}
