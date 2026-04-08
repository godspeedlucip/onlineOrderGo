package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/product/domain"
)

type flakyInvalidator struct {
	failOnce bool
}

func (f *flakyInvalidator) InvalidateCategory(ctx context.Context, categoryID int64) error {
	return nil
}
func (f *flakyInvalidator) InvalidateDish(ctx context.Context, dishID int64, categoryID int64) error {
	if f.failOnce {
		f.failOnce = false
		return errors.New("temporary redis failure")
	}
	return nil
}
func (f *flakyInvalidator) InvalidateSetmeal(ctx context.Context, setmealID int64, categoryID int64) error {
	return nil
}
func (f *flakyInvalidator) InvalidateByCategory(ctx context.Context, categoryID int64) error {
	return nil
}

func TestRedisInvalidationOutbox_RetryAfterFailure(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	ctx := context.Background()
	key := "product:test:outbox"
	outbox := NewRedisInvalidationOutbox(client, key)
	if err := outbox.Enqueue(ctx, domain.CacheInvalidateTask{
		Operation:   "dish",
		CategoryID:  8,
		EntityID:    77,
		EnqueueAtMS: time.Now().Add(-time.Second).UnixMilli(),
	}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	invalidator := &flakyInvalidator{failOnce: true}
	processed, err := outbox.RunOnce(ctx, invalidator, 10)
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if processed != 0 {
		t.Fatalf("first run should not process success, got=%d", processed)
	}
	if c := mini.ZCard(key); c != 1 {
		t.Fatalf("expected one pending retry, got=%d", c)
	}

	mini.FastForward(3 * time.Second)
	processed, err = outbox.RunOnce(ctx, invalidator, 10)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if processed != 1 {
		t.Fatalf("second run should process one item, got=%d", processed)
	}
	if c := mini.ZCard(key); c != 0 {
		t.Fatalf("expected empty outbox after retry success, got=%d", c)
	}
}
