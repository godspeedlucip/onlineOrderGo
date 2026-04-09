package idempotency

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisStore(client redis.UniversalClient, prefix string) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: strings.TrimSpace(prefix),
	}
}

func (s *RedisStore) TryAcquire(ctx context.Context, messageID string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = time.Minute
	}
	key := s.key(messageID)
	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *RedisStore) key(messageID string) string {
	if s.prefix == "" {
		return "websocket_push:dedupe:" + messageID
	}
	return s.prefix + ":dedupe:" + messageID
}
