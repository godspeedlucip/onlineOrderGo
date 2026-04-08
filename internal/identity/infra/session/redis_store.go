package session

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/identity/domain"
)

type RedisStore struct {
	client redis.UniversalClient
	prefix string
}

var compareAndIncreaseScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then
	current = '1'
	redis.call('SET', KEYS[1], current)
end
if tonumber(current) ~= tonumber(ARGV[1]) then
	return {tonumber(current), 0}
end
local next = tonumber(current) + 1
redis.call('SET', KEYS[1], tostring(next))
return {next, 1}
`)

func NewRedisStore(client redis.UniversalClient, keyPrefix string) *RedisStore {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "identity:session"
	}
	return &RedisStore{client: client, prefix: prefix}
}

func (s *RedisStore) EnsureVersion(ctx context.Context, accountType domain.AccountType, accountID int64) (int64, error) {
	if s == nil || s.client == nil {
		return 0, fmt.Errorf("redis client is not initialized")
	}
	key := s.versionKey(accountType, accountID)
	ok, err := s.client.SetNX(ctx, key, "1", 0).Result()
	if err != nil {
		return 0, err
	}
	if ok {
		return 1, nil
	}
	return s.readVersion(ctx, key)
}

func (s *RedisStore) GetVersion(ctx context.Context, accountType domain.AccountType, accountID int64) (int64, bool, error) {
	if s == nil || s.client == nil {
		return 0, false, fmt.Errorf("redis client is not initialized")
	}
	key := s.versionKey(accountType, accountID)
	v, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}

func (s *RedisStore) CompareAndIncreaseVersion(ctx context.Context, accountType domain.AccountType, accountID int64, expected int64) (int64, bool, error) {
	if s == nil || s.client == nil {
		return 0, false, fmt.Errorf("redis client is not initialized")
	}
	key := s.versionKey(accountType, accountID)
	raw, err := compareAndIncreaseScript.Run(ctx, s.client, []string{key}, expected).Result()
	if err != nil {
		return 0, false, err
	}

	arr, ok := raw.([]any)
	if !ok || len(arr) < 2 {
		return 0, false, fmt.Errorf("invalid cas response")
	}
	newVersion, err := toInt64(arr[0])
	if err != nil {
		return 0, false, err
	}
	updated, err := toInt64(arr[1])
	if err != nil {
		return 0, false, err
	}
	return newVersion, updated == 1, nil
}

func (s *RedisStore) MarkTokenRevoked(ctx context.Context, tokenID string, expireAt time.Time) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	if tokenID == "" {
		return nil
	}

	ttl := time.Until(expireAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	return s.client.Set(ctx, s.revokedTokenKey(tokenID), "1", ttl).Err()
}

func (s *RedisStore) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is not initialized")
	}
	if tokenID == "" {
		return false, nil
	}

	exists, err := s.client.Exists(ctx, s.revokedTokenKey(tokenID)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisStore) versionKey(accountType domain.AccountType, accountID int64) string {
	return fmt.Sprintf("%s:version:%s:%d", s.prefix, accountType, accountID)
}

func (s *RedisStore) revokedTokenKey(tokenID string) string {
	return fmt.Sprintf("%s:revoked:%s", s.prefix, tokenID)
}

func (s *RedisStore) readVersion(ctx context.Context, key string) (int64, error) {
	v, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func toInt64(v any) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, fmt.Errorf("not int64: %T", v)
	}
}
