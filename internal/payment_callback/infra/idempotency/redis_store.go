package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

type idemRecord struct {
	Token     string `json:"token"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
	UpdatedAt int64  `json:"updated_at_unix_ms"`
}

var markDoneScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then return 0 end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.state ~= 'PROCESSING' then return 0 end
obj.state = 'DONE'
obj.reason = ''
obj.updated_at_unix_ms = tonumber(ARGV[2])
redis.call('SET', KEYS[1], cjson.encode(obj), 'KEEPTTL')
return 1
`)

var markFailedScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then return 0 end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.state ~= 'PROCESSING' then return 0 end
redis.call('DEL', KEYS[1])
return 1
`)

func NewRedisStore(client redis.UniversalClient, keyPrefix string) *RedisStore {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "payment_callback:idempotency"
	}
	return &RedisStore{client: client, keyPrefix: prefix}
}

func (s *RedisStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, fmt.Errorf("redis client is not initialized")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	rec := idemRecord{
		Token:     token,
		State:     "PROCESSING",
		UpdatedAt: time.Now().UnixMilli(),
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return "", false, err
	}
	ok, err := s.client.SetNX(ctx, s.buildKey(scene, key), raw, ttl).Result()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return token, true, nil
}

func (s *RedisStore) MarkDone(ctx context.Context, scene, key, token string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	_, err := markDoneScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token, time.Now().UnixMilli()).Result()
	return err
}

func (s *RedisStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = reason
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	_, err := markFailedScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token).Result()
	return err
}

func (s *RedisStore) buildKey(scene, key string) string {
	return fmt.Sprintf("%s:%s:%s", s.keyPrefix, scene, key)
}
