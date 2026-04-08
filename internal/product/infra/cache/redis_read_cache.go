package cache

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/product/domain"
)

type RedisReadCache struct {
	client    redis.UniversalClient
	namespace string
}

func NewRedisReadCache(client redis.UniversalClient, namespace string) *RedisReadCache {
	return &RedisReadCache{client: client, namespace: strings.TrimSpace(namespace)}
}

func (c *RedisReadCache) GetCategories(ctx context.Context, key string) ([]domain.CategoryVO, bool, error) {
	var out []domain.CategoryVO
	ok, err := c.get(ctx, key, &out)
	if out == nil {
		out = make([]domain.CategoryVO, 0)
	}
	return out, ok, err
}

func (c *RedisReadCache) SetCategories(ctx context.Context, key string, value []domain.CategoryVO, ttl time.Duration) error {
	return c.set(ctx, key, value, ttl)
}

func (c *RedisReadCache) GetDishes(ctx context.Context, key string) ([]domain.DishVO, bool, error) {
	var out []domain.DishVO
	ok, err := c.get(ctx, key, &out)
	if out == nil {
		out = make([]domain.DishVO, 0)
	}
	return out, ok, err
}

func (c *RedisReadCache) SetDishes(ctx context.Context, key string, value []domain.DishVO, ttl time.Duration) error {
	return c.set(ctx, key, value, ttl)
}

func (c *RedisReadCache) GetSetmeals(ctx context.Context, key string) ([]domain.SetmealVO, bool, error) {
	var out []domain.SetmealVO
	ok, err := c.get(ctx, key, &out)
	if out == nil {
		out = make([]domain.SetmealVO, 0)
	}
	return out, ok, err
}

func (c *RedisReadCache) SetSetmeals(ctx context.Context, key string, value []domain.SetmealVO, ttl time.Duration) error {
	return c.set(ctx, key, value, ttl)
}

func (c *RedisReadCache) get(ctx context.Context, key string, out any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	raw, err := c.client.Get(ctx, c.buildKey(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisReadCache) set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.buildKey(key), payload, ttl).Err()
}

func (c *RedisReadCache) deleteByPrefix(prefix string) error {
	if c == nil || c.client == nil {
		return nil
	}
	ctx := context.Background()
	pattern := c.buildKey(prefix) + "*"
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}

func (c *RedisReadCache) buildKey(key string) string {
	if c.namespace == "" {
		return key
	}
	return c.namespace + ":" + key
}
