package idempotency

import (
	"context"
	"sync"
	"time"
)

type idemStatus string

const (
	statusProcessing idemStatus = "PROCESSING"
	statusDone       idemStatus = "DONE"
	statusFailed     idemStatus = "FAILED"
)

type record struct {
	token     string
	status    idemStatus
	result    []byte
	expireAt  time.Time
	updatedAt time.Time
}

type InMemoryStore struct {
	mu    sync.Mutex
	items map[string]record
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]record)}
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

	s.items[mapKey] = record{token: token, status: statusProcessing, expireAt: now.Add(ttl), updatedAt: now}
	return token, true, nil
}

func (s *InMemoryStore) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
	_ = ctx
	mapKey := scene + ":" + key
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.items[mapKey]
	if !ok || old.token != token {
		return nil
	}
	if old.status != statusProcessing {
		return nil
	}
	old.status = statusDone
	if len(result) > 0 {
		old.result = append([]byte(nil), result...)
	} else {
		old.result = nil
	}
	old.updatedAt = now
	s.items[mapKey] = old
	return nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = ctx
	_ = reason
	mapKey := scene + ":" + key
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.items[mapKey]
	if !ok || old.token != token {
		return nil
	}
	if old.status != statusProcessing {
		return nil
	}
	old.status = statusFailed
	old.result = nil
	old.updatedAt = now
	// Failed record is removed to allow immediate retry.
	delete(s.items, mapKey)
	return nil
}

func (s *InMemoryStore) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	_ = ctx
	mapKey := scene + ":" + key
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.items[mapKey]
	if !ok {
		return nil, false, nil
	}
	if now.After(old.expireAt) {
		delete(s.items, mapKey)
		return nil, false, nil
	}
	if old.status != statusDone {
		return nil, false, nil
	}
	return append([]byte(nil), old.result...), true, nil
}
