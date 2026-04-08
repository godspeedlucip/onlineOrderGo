package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisReportCache struct {
	client    redis.UniversalClient
	namespace string
}

func NewRedisReportCache(client redis.UniversalClient, namespace string) *RedisReportCache {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "report:cache"
	}
	return &RedisReportCache{client: client, namespace: ns}
}

func (c *RedisReportCache) Get(ctx context.Context, key string, out any) (bool, error) {
	if c == nil || c.client == nil {
		return false, fmt.Errorf("redis client is not initialized")
	}
	raw, err := c.client.Get(ctx, c.buildKey(key)).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisReportCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.buildKey(key), raw, ttl).Err()
}

func (c *RedisReportCache) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", c.namespace, strings.TrimSpace(key))
}
