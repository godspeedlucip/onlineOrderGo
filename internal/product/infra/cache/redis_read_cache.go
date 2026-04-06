package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type cacheEntry struct {
	Payload  []byte
	ExpireAt time.Time
}

type RedisReadCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewRedisReadCache() *RedisReadCache {
	return &RedisReadCache{entries: make(map[string]cacheEntry)}
}

func (c *RedisReadCache) GetCategories(ctx context.Context, key string) ([]domain.CategoryVO, bool, error) {
	_ = ctx
	var out []domain.CategoryVO
	ok, err := c.get(key, &out)
	return out, ok, err
}

func (c *RedisReadCache) SetCategories(ctx context.Context, key string, value []domain.CategoryVO, ttl time.Duration) error {
	_ = ctx
	return c.set(key, value, ttl)
}

func (c *RedisReadCache) GetDishes(ctx context.Context, key string) ([]domain.DishVO, bool, error) {
	_ = ctx
	var out []domain.DishVO
	ok, err := c.get(key, &out)
	return out, ok, err
}

func (c *RedisReadCache) SetDishes(ctx context.Context, key string, value []domain.DishVO, ttl time.Duration) error {
	_ = ctx
	return c.set(key, value, ttl)
}

func (c *RedisReadCache) GetSetmeals(ctx context.Context, key string) ([]domain.SetmealVO, bool, error) {
	_ = ctx
	var out []domain.SetmealVO
	ok, err := c.get(key, &out)
	return out, ok, err
}

func (c *RedisReadCache) SetSetmeals(ctx context.Context, key string, value []domain.SetmealVO, ttl time.Duration) error {
	_ = ctx
	return c.set(key, value, ttl)
}

func (c *RedisReadCache) get(key string, out any) (bool, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if now.After(entry.ExpireAt) {
		c.mu.Lock()
		current, still := c.entries[key]
		if still && now.After(current.ExpireAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return false, nil
	}

	if err := json.Unmarshal(entry.Payload, out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisReadCache) set(key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	entry := cacheEntry{Payload: payload, ExpireAt: time.Now().Add(ttl)}

	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.entries[key]; ok {
		// Conditional update: skip write when value unchanged and old entry still alive.
		if time.Now().Before(old.ExpireAt) && bytes.Equal(old.Payload, payload) {
			return nil
		}
	}
	c.entries[key] = entry
	return nil
}

// TODO: replace in-memory cache with Redis implementation while preserving key/TTL semantics.
func (c *RedisReadCache) deleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if len(prefix) == 0 || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			delete(c.entries, k)
		}
	}
}