package lock

import (
	"context"
	"sync"
	"time"
)

type rec struct {
	token    string
	expireAt time.Time
}

type InMemoryLocker struct {
	mu    sync.Mutex
	items map[string]rec
}

func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{items: make(map[string]rec)}
}

func (l *InMemoryLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (func() error, bool, error) {
	_ = ctx
	if key == "" {
		return func() error { return nil }, false, nil
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	now := time.Now()
	token := now.Format(time.RFC3339Nano)
	l.mu.Lock()
	if old, ok := l.items[key]; ok && now.Before(old.expireAt) {
		l.mu.Unlock()
		return func() error { return nil }, false, nil
	}
	l.items[key] = rec{token: token, expireAt: now.Add(ttl)}
	l.mu.Unlock()
	unlock := func() error {
		l.mu.Lock()
		defer l.mu.Unlock()
		old, ok := l.items[key]
		if !ok {
			return nil
		}
		if old.token != token {
			return nil
		}
		delete(l.items, key)
		return nil
	}
	return unlock, true, nil
}
