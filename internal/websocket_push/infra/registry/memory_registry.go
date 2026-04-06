package registry

import (
	"context"
	"sync"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type MemoryRegistry struct {
	mu       sync.RWMutex
	sessions map[string]domain.Session
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{sessions: make(map[string]domain.Session)}
}

func (r *MemoryRegistry) Add(ctx context.Context, s domain.Session) error {
	_ = ctx
	if s.SessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	r.mu.Lock()
	if _, exists := r.sessions[s.SessionID]; exists {
		r.mu.Unlock()
		return domain.NewBizError(domain.CodeConflict, "session already exists", nil)
	}
	r.sessions[s.SessionID] = s
	r.mu.Unlock()
	return nil
}

func (r *MemoryRegistry) Remove(ctx context.Context, sessionID string) error {
	_ = ctx
	r.mu.Lock()
	delete(r.sessions, sessionID)
	r.mu.Unlock()
	return nil
}

func (r *MemoryRegistry) Touch(ctx context.Context, sessionID string) error {
	_ = ctx
	r.mu.Lock()
	old, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return domain.NewBizError(domain.CodeConflict, "session not found", nil)
	}
	old.LastSeenAt = time.Now()
	r.sessions[sessionID] = old
	r.mu.Unlock()
	return nil
}

func (r *MemoryRegistry) GetByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	cp := v
	return &cp, nil
}

func (r *MemoryRegistry) FindByUser(ctx context.Context, userID int64) ([]domain.Session, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Session, 0)
	for _, s := range r.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryRegistry) FindByShop(ctx context.Context, shopID int64) ([]domain.Session, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Session, 0)
	for _, s := range r.sessions {
		if s.ShopID == shopID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryRegistry) FindByChannel(ctx context.Context, channel string) ([]domain.Session, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Session, 0)
	for _, s := range r.sessions {
		if hasChannel(s.Channels, channel) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryRegistry) FindAll(ctx context.Context) ([]domain.Session, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, s)
	}
	return out, nil
}

func hasChannel(channels []string, target string) bool {
	for _, c := range channels {
		if c == target {
			return true
		}
	}
	return false
}
