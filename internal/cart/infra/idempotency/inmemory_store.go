package idempotency

import (
	"context"
	"sync"
	"time"
)

type status string

const (
	statusProcessing status = "PROCESSING"
	statusDone       status = "DONE"
)

type rec struct {
	token     string
	status    status
	expireAt  time.Time
	updatedAt time.Time
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
		ttl = 5 * time.Minute
	}
	mapKey := scene + ":" + key
	now := time.Now()
	token := now.Format(time.RFC3339Nano)

	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.items[mapKey]; ok {
		if now.After(old.expireAt) {
			delete(s.items, mapKey)
		} else {
			return old.token, false, nil
		}
	}
	s.items[mapKey] = rec{token: token, status: statusProcessing, expireAt: now.Add(ttl), updatedAt: now}
	return token, true, nil
}

func (s *InMemoryStore) MarkDone(ctx context.Context, scene, key, token string) error {
	_ = ctx
	mapKey := scene + ":" + key
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[mapKey]
	if !ok || old.token != token || old.status != statusProcessing {
		return nil
	}
	old.status = statusDone
	old.updatedAt = now
	s.items[mapKey] = old
	return nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	_ = reason
	mapKey := scene + ":" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[mapKey]
	if !ok || old.token != token {
		return nil
	}
	delete(s.items, mapKey)
	return nil
}