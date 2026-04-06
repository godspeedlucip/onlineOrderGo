package idempotency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-baseline-skeleton/internal/baseline/domain"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]*domain.IdempotencyRecord
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]*domain.IdempotencyRecord)}
}

func (s *InMemoryStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	_ = ctx
	now := time.Now()
	mapKey := buildKey(scene, key)
	token := fmt.Sprintf("%d", now.UnixNano())

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, exists := s.items[mapKey]
	if !exists || now.After(rec.ExpireAt) {
		s.items[mapKey] = &domain.IdempotencyRecord{
			Scene:     scene,
			Key:       key,
			Token:     token,
			Status:    domain.StatusProcessing,
			UpdatedAt: now,
			ExpireAt:  now.Add(ttl),
		}
		return token, true, nil
	}

	if rec.Status == domain.StatusFailed {
		// FAILED -> PROCESSING with a new token.
		rec.Token = token
		rec.Status = domain.StatusProcessing
		rec.Reason = ""
		rec.Payload = nil
		rec.UpdatedAt = now
		rec.ExpireAt = now.Add(ttl)
		return token, true, nil
	}

	return rec.Token, false, nil
}

func (s *InMemoryStore) Get(ctx context.Context, scene, key string) (*domain.IdempotencyRecord, error) {
	_ = ctx
	mapKey := buildKey(scene, key)
	now := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.items[mapKey]
	if !ok || now.After(rec.ExpireAt) {
		return nil, nil
	}
	cloned := *rec
	return &cloned, nil
}

func (s *InMemoryStore) MarkSuccess(ctx context.Context, scene, key, token string, payload []byte) (bool, error) {
	_ = ctx
	mapKey := buildKey(scene, key)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.items[mapKey]
	if !ok {
		return false, nil
	}
	if rec.Token != token || rec.Status != domain.StatusProcessing {
		return false, nil
	}
	rec.Status = domain.StatusSucceeded
	rec.Payload = payload
	rec.Reason = ""
	rec.UpdatedAt = time.Now()
	return true, nil
}

func (s *InMemoryStore) MarkFailed(ctx context.Context, scene, key, token, reason string) (bool, error) {
	_ = ctx
	mapKey := buildKey(scene, key)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.items[mapKey]
	if !ok {
		return false, nil
	}
	if rec.Token != token || rec.Status != domain.StatusProcessing {
		return false, nil
	}
	rec.Status = domain.StatusFailed
	rec.Payload = nil
	rec.Reason = reason
	rec.UpdatedAt = time.Now()
	return true, nil
}

func buildKey(scene, key string) string {
	return scene + ":" + key
}