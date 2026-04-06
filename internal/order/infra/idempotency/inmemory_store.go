package idempotency

import (
	"context"
	"sync"
	"time"
)

type item struct {
	token    string
	expireAt time.Time
	state    string
	reason   string
}

type InMemoryStore struct {
	mu    sync.Mutex
	items map[string]item
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: map[string]item{}}
}

func (s *InMemoryStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now()
	mapKey := scene + ":" + key
	token := now.Format(time.RFC3339Nano)

	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.items[mapKey]; ok && now.Before(old.expireAt) {
		return old.token, false, nil
	}
	s.items[mapKey] = item{token: token, expireAt: now.Add(ttl), state: "PROCESSING"}
	return token, true, nil
}

func (s *InMemoryStore) MarkDone(ctx context.Context, scene, key, token string) error {
	_ = ctx
	mapKey := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[mapKey]
	if !ok {
		return nil
	}
	if old.token != token {
		return nil
	}
	old.state = "DONE"
	old.reason = ""
	s.items[mapKey] = old
	return nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	mapKey := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[mapKey]
	if !ok {
		return nil
	}
	if old.token != token {
		return nil
	}
	old.state = "FAILED"
	old.reason = reason
	// Failed command should be retryable.
	delete(s.items, mapKey)
	return nil
}
