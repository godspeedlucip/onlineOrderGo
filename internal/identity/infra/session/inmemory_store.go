package session

import (
	"context"
	"strconv"
	"sync"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type InMemoryStore struct {
	mu            sync.RWMutex
	versionByUser map[string]int64
	revokedToken  map[string]time.Time
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		versionByUser: make(map[string]int64),
		revokedToken:  make(map[string]time.Time),
	}
}

func (s *InMemoryStore) EnsureVersion(ctx context.Context, accountType domain.AccountType, accountID int64) (int64, error) {
	_ = ctx
	key := sessionKey(accountType, accountID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.versionByUser[key]; ok {
		return v, nil
	}
	s.versionByUser[key] = 1
	return 1, nil
}

func (s *InMemoryStore) GetVersion(ctx context.Context, accountType domain.AccountType, accountID int64) (int64, bool, error) {
	_ = ctx
	key := sessionKey(accountType, accountID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versionByUser[key]
	return v, ok, nil
}

func (s *InMemoryStore) CompareAndIncreaseVersion(ctx context.Context, accountType domain.AccountType, accountID int64, expected int64) (int64, bool, error) {
	_ = ctx
	key := sessionKey(accountType, accountID)
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.versionByUser[key]
	if !ok {
		current = 1
		s.versionByUser[key] = current
	}
	if current != expected {
		return current, false, nil
	}
	next := current + 1
	s.versionByUser[key] = next
	return next, true, nil
}

func (s *InMemoryStore) MarkTokenRevoked(ctx context.Context, tokenID string, expireAt time.Time) error {
	_ = ctx
	if tokenID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedToken[tokenID] = expireAt
	return nil
}

func (s *InMemoryStore) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	_ = ctx
	if tokenID == "" {
		return false, nil
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	expireAt, ok := s.revokedToken[tokenID]
	if !ok {
		return false, nil
	}
	if now.After(expireAt) {
		delete(s.revokedToken, tokenID)
		return false, nil
	}
	return true, nil
}

func sessionKey(accountType domain.AccountType, accountID int64) string {
	return string(accountType) + ":" + strconv.FormatInt(accountID, 10)
}