package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type entry struct {
	payload  []byte
	expireAt time.Time
}

type RedisReportCache struct {
	mu    sync.RWMutex
	items map[string]entry
}

func NewRedisReportCache() *RedisReportCache {
	return &RedisReportCache{items: make(map[string]entry)}
}

func (c *RedisReportCache) Get(ctx context.Context, key string, out any) (bool, error) {
	_ = ctx
	now := time.Now()
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if now.After(e.expireAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return false, nil
	}
	if err := json.Unmarshal(e.payload, out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisReportCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	_ = ctx
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.items[key] = entry{payload: b, expireAt: time.Now().Add(ttl)}
	c.mu.Unlock()
	return nil
}

// TODO: replace with real Redis implementation and key namespace.