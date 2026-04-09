package idempotency

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]time.Time)}
}

func (s *MemoryStore) TryAcquire(ctx context.Context, messageID string, ttl time.Duration) (bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = time.Minute
	}
	now := time.Now()
	expireBefore := now.Add(-ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, ts := range s.items {
		if ts.Before(expireBefore) {
			delete(s.items, key)
		}
	}
	if _, ok := s.items[messageID]; ok {
		return false, nil
	}
	s.items[messageID] = now
	return true, nil
}
