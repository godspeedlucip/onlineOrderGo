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
	result   []byte
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

func (s *InMemoryStore) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
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
	if len(result) > 0 {
		old.result = append([]byte(nil), result...)
	}
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
	delete(s.items, mapKey)
	return nil
}

func (s *InMemoryStore) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	_ = ctx
	mapKey := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[mapKey]
	if !ok {
		return nil, false, nil
	}
	if old.state != "DONE" {
		return nil, false, nil
	}
	if len(old.result) == 0 {
		return nil, true, nil
	}
	return append([]byte(nil), old.result...), true, nil
}
