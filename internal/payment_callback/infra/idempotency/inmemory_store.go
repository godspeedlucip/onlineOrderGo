package idempotency

import (
	"context"
	"sync"
	"time"
)

type rec struct {
	token    string
	expireAt time.Time
	state    string
	reason   string
}

type InMemoryStore struct {
	mu    sync.Mutex
	items map[string]rec
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]rec)}
}

func (s *InMemoryStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	mapKey := scene + ":" + key
	now := time.Now()
	token := now.Format(time.RFC3339Nano)

	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.items[mapKey]; ok && now.Before(old.expireAt) {
		return old.token, false, nil
	}
	s.items[mapKey] = rec{token: token, expireAt: now.Add(ttl), state: "PROCESSING"}
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
	// keep dedupe window for repeated provider retries.
	s.items[mapKey] = old
	return nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	mapKey := scene + ":" + key
	s.mu.Lock()
	old, ok := s.items[mapKey]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	if old.token != token {
		s.mu.Unlock()
		return nil
	}
	old.state = "FAILED"
	old.reason = reason
	// failed callback should be retryable.
	delete(s.items, mapKey)
	s.mu.Unlock()
	return nil
}
