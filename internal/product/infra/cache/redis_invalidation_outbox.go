package cache

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/product/domain"
)

type RedisInvalidationOutbox struct {
	client redis.UniversalClient
	key    string
}

func NewRedisInvalidationOutbox(client redis.UniversalClient, key string) *RedisInvalidationOutbox {
	k := strings.TrimSpace(key)
	if k == "" {
		k = "product:cache_invalidation:outbox"
	}
	return &RedisInvalidationOutbox{client: client, key: k}
}

func (o *RedisInvalidationOutbox) Enqueue(ctx context.Context, task domain.CacheInvalidateTask) error {
	if o == nil || o.client == nil {
		return nil
	}
	if task.EnqueueAtMS <= 0 {
		task.EnqueueAtMS = time.Now().UnixMilli()
	}
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	score := float64(task.EnqueueAtMS)
	return o.client.ZAdd(ctx, o.key, redis.Z{Score: score, Member: string(payload)}).Err()
}

func (o *RedisInvalidationOutbox) RunOnce(ctx context.Context, invalidator domain.ProductCacheInvalidator, limit int) (int, error) {
	if o == nil || o.client == nil || invalidator == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	now := time.Now().UnixMilli()
	items, err := o.client.ZRangeByScore(ctx, o.key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now, 10),
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, member := range items {
		removed, remErr := o.client.ZRem(ctx, o.key, member).Result()
		if remErr != nil || removed == 0 {
			if remErr != nil {
				return processed, remErr
			}
			continue
		}

		var task domain.CacheInvalidateTask
		if err := json.Unmarshal([]byte(member), &task); err != nil {
			continue
		}
		if err := runInvalidationTask(ctx, invalidator, task); err != nil {
			task.RetryCount++
			task.EnqueueAtMS = now + retryDelayMS(task.RetryCount)
			_ = o.Enqueue(ctx, task)
			continue
		}
		processed++
	}
	return processed, nil
}

func runInvalidationTask(ctx context.Context, invalidator domain.ProductCacheInvalidator, task domain.CacheInvalidateTask) error {
	switch task.Operation {
	case "category":
		return invalidator.InvalidateCategory(ctx, task.EntityID)
	case "dish":
		return invalidator.InvalidateDish(ctx, task.EntityID, task.CategoryID)
	case "setmeal":
		return invalidator.InvalidateSetmeal(ctx, task.EntityID, task.CategoryID)
	case "by_category":
		return invalidator.InvalidateByCategory(ctx, task.CategoryID)
	default:
		return domain.NewBizError(domain.CodeInvalidArgument, "unknown cache invalidation operation", nil)
	}
}

func retryDelayMS(retry int) int64 {
	if retry < 1 {
		return 1000
	}
	seconds := int64(math.Pow(2, float64(retry)))
	if seconds > 60 {
		seconds = 60
	}
	return seconds * 1000
}
