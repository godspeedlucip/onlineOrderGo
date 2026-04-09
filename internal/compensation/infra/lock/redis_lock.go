package lock

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var releaseLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`)

type RedisLocker struct {
	client    redis.UniversalClient
	keyPrefix string
}

func NewRedisLocker(client redis.UniversalClient, keyPrefix string) *RedisLocker {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "compensation:lock"
	}
	return &RedisLocker{client: client, keyPrefix: prefix}
}

func (l *RedisLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (func() error, bool, error) {
	if l == nil || l.client == nil {
		return nil, false, fmt.Errorf("redis client is not initialized")
	}
	lockKey := l.buildKey(key)
	if lockKey == "" {
		return func() error { return nil }, false, nil
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	ok, err := l.client.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return func() error { return nil }, false, nil
	}
	return func() error {
		_, err := releaseLockScript.Run(context.Background(), l.client, []string{lockKey}, token).Result()
		return err
	}, true, nil
}

func (l *RedisLocker) buildKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return l.keyPrefix + ":" + key
}
