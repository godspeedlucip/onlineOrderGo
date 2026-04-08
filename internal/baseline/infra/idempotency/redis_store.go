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

	"go-baseline-skeleton/internal/baseline/domain"
)

type RedisStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

type redisRecord struct {
	Scene      string                   `json:"scene"`
	Key        string                   `json:"key"`
	Token      string                   `json:"token"`
	Status     domain.IdempotencyStatus `json:"status"`
	PayloadB64 string                   `json:"payload_b64,omitempty"`
	Reason     string                   `json:"reason,omitempty"`
	UpdatedAt  int64                    `json:"updated_at_unix_ms"`
	ExpireAt   int64                    `json:"expire_at_unix_ms"`
}

var transitionFailedToProcessingScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local obj = cjson.decode(raw)
if obj.status ~= 'FAILED' then
	return 0
end
obj.token = ARGV[1]
obj.status = 'PROCESSING'
obj.payload_b64 = ''
obj.reason = ''
obj.updated_at_unix_ms = tonumber(ARGV[2])
obj.expire_at_unix_ms = tonumber(ARGV[3])
redis.call('SET', KEYS[1], cjson.encode(obj), 'EX', ARGV[4])
return 1
`)

var markSuccessScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.status ~= 'PROCESSING' then
	return 0
end
obj.status = 'SUCCEEDED'
obj.payload_b64 = ARGV[2]
obj.reason = ''
obj.updated_at_unix_ms = tonumber(ARGV[3])
redis.call('SET', KEYS[1], cjson.encode(obj), 'KEEPTTL')
return 1
`)

var markFailedScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local obj = cjson.decode(raw)
if obj.token ~= ARGV[1] or obj.status ~= 'PROCESSING' then
	return 0
end
obj.status = 'FAILED'
obj.payload_b64 = ''
obj.reason = ARGV[2]
obj.updated_at_unix_ms = tonumber(ARGV[3])
redis.call('SET', KEYS[1], cjson.encode(obj), 'KEEPTTL')
return 1
`)

func NewRedisStore(client redis.UniversalClient, keyPrefix string) *RedisStore {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "baseline:idempotency"
	}
	return &RedisStore{client: client, keyPrefix: prefix}
}

func NewRedisClient(addr, password string, db int) redis.UniversalClient {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func (s *RedisStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, fmt.Errorf("redis client is not initialized")
	}

	ttlSeconds := int64(ttl.Seconds())
	if ttlSeconds <= 0 {
		ttlSeconds = 1
	}

	now := time.Now()
	token := strconv.FormatInt(now.UnixNano(), 10)
	rec := redisRecord{
		Scene:     scene,
		Key:       key,
		Token:     token,
		Status:    domain.StatusProcessing,
		UpdatedAt: now.UnixMilli(),
		ExpireAt:  now.Add(ttl).UnixMilli(),
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return "", false, err
	}

	redisKey := s.buildKey(scene, key)
	ok, err := s.client.SetNX(ctx, redisKey, payload, ttl).Result()
	if err != nil {
		return "", false, err
	}
	if ok {
		return token, true, nil
	}

	updatedRaw, err := transitionFailedToProcessingScript.Run(ctx, s.client, []string{redisKey}, token, now.UnixMilli(), now.Add(ttl).UnixMilli(), ttlSeconds).Result()
	if err != nil {
		return "", false, err
	}
	updated := toInt64(updatedRaw) == 1
	if updated {
		return token, true, nil
	}

	return "", false, nil
}

func (s *RedisStore) Get(ctx context.Context, scene, key string) (*domain.IdempotencyRecord, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("redis client is not initialized")
	}

	raw, err := s.client.Get(ctx, s.buildKey(scene, key)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var rec redisRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, err
	}

	payload, err := decodePayload(rec.PayloadB64)
	if err != nil {
		return nil, err
	}

	return &domain.IdempotencyRecord{
		Scene:     rec.Scene,
		Key:       rec.Key,
		Token:     rec.Token,
		Status:    rec.Status,
		Payload:   payload,
		Reason:    rec.Reason,
		UpdatedAt: time.UnixMilli(rec.UpdatedAt),
		ExpireAt:  time.UnixMilli(rec.ExpireAt),
	}, nil
}

func (s *RedisStore) MarkSuccess(ctx context.Context, scene, key, token string, payload []byte) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is not initialized")
	}

	updatedRaw, err := markSuccessScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token, base64.StdEncoding.EncodeToString(payload), time.Now().UnixMilli()).Result()
	if err != nil {
		return false, err
	}
	return toInt64(updatedRaw) == 1, nil
}

func (s *RedisStore) MarkFailed(ctx context.Context, scene, key, token, reason string) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is not initialized")
	}

	updatedRaw, err := markFailedScript.Run(ctx, s.client, []string{s.buildKey(scene, key)}, token, reason, time.Now().UnixMilli()).Result()
	if err != nil {
		return false, err
	}
	return toInt64(updatedRaw) == 1, nil
}

func (s *RedisStore) buildKey(scene, key string) string {
	return fmt.Sprintf("%s:%s:%s", s.keyPrefix, scene, key)
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case uint64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

func decodePayload(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, nil
	}
	payload, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
