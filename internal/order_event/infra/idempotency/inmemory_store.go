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
	mu   sync.Mutex
	data map[string]rec
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]rec)}
}

func (s *InMemoryStore) Acquire(ctx context.Context, eventID string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	now := time.Now()
	token := now.Format(time.RFC3339Nano)
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.data[eventID]; ok && now.Before(old.expireAt) {
		return old.token, false, nil
	}
	s.data[eventID] = rec{token: token, expireAt: now.Add(ttl), state: "PROCESSING"}
	return token, true, nil
}

func (s *InMemoryStore) MarkDone(ctx context.Context, eventID, token string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.data[eventID]
	if !ok || old.token != token {
		return nil
	}
	old.state = "DONE"
	old.reason = ""
	s.data[eventID] = old
	return nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, eventID, token, reason string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.data[eventID]
	if !ok || old.token != token {
		return nil
	}
	old.state = "FAILED"
	old.reason = reason
	delete(s.data, eventID)
	return nil
}
