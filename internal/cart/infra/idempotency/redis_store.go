package idempotency

import (
	"context"
	"encoding/base64"
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

type redisRecord struct {
	Scene     string `json:"scene"`
	Key       string `json:"key"`
	Token     string `json:"token"`
	Status    string `json:"status"`
	ResultB64 string `json:"result_b64,omitempty"`
	UpdatedAt int64  `json:"updated_at_unix_ms"`
	ExpireAt  int64  `json:"expire_at_unix_ms"`
}

var cartMarkDoneScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.status ~= 'PROCESSING' then
	return 0
end
obj.status = 'DONE'
obj.result_b64 = ARGV[2]
obj.updated_at_unix_ms = tonumber(ARGV[3])
redis.call('SET', KEYS[1], cjson.encode(obj), 'KEEPTTL')
return 1
`)

var cartMarkFailedScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.status ~= 'PROCESSING' then
	return 0
end
redis.call('DEL', KEYS[1])
return 1
`)

func NewRedisStore(client redis.UniversalClient, keyPrefix string) *RedisStore {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "cart:idempotency"
	}
	return &RedisStore{client: client, keyPrefix: prefix}
}

func (s *RedisStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, fmt.Errorf("redis client is not initialized")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now()
	token := strconv.FormatInt(now.UnixNano(), 10)
	rec := redisRecord{
		Scene:     scene,
		Key:       key,
		Token:     token,
		Status:    "PROCESSING",
		UpdatedAt: now.UnixMilli(),
		ExpireAt:  now.Add(ttl).UnixMilli(),
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

func (s *RedisStore) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	encoded := ""
	if len(result) > 0 {
		encoded = base64.StdEncoding.EncodeToString(result)
	}
	_, err := cartMarkDoneScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token, encoded, time.Now().UnixMilli()).Result()
	return err
}

func (s *RedisStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	_ = reason
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	_, err := cartMarkFailedScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token).Result()
	return err
}

func (s *RedisStore) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	if s == nil || s.client == nil {
		return nil, false, fmt.Errorf("redis client is not initialized")
	}
	raw, err := s.client.Get(ctx, s.buildKey(scene, key)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var rec redisRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, false, err
	}
	if rec.Status != "DONE" {
		return nil, false, nil
	}
	if strings.TrimSpace(rec.ResultB64) == "" {
		return nil, true, nil
	}
	payload, err := base64.StdEncoding.DecodeString(rec.ResultB64)
	if err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

func (s *RedisStore) buildKey(scene, key string) string {
	return fmt.Sprintf("%s:%s:%s", s.keyPrefix, scene, key)
}
