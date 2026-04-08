package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/product/domain"
)

func TestRedisReadCache_SetGetAndTTL(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	cache := NewRedisReadCache(client, "product:test")
	ctx := context.Background()
	key := "product:category:type=1:status=1:client=user"

	value := []domain.CategoryVO{{ID: 1, Name: "Hot", Type: 1}}
	if err := cache.SetCategories(ctx, key, value, 2*time.Second); err != nil {
		t.Fatalf("SetCategories failed: %v", err)
	}
	got, ok, err := cache.GetCategories(ctx, key)
	if err != nil || !ok || len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("GetCategories mismatch: ok=%v err=%v got=%+v", ok, err, got)
	}

	mini.FastForward(3 * time.Second)
	_, ok, err = cache.GetCategories(ctx, key)
	if err != nil {
		t.Fatalf("GetCategories after ttl failed: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss after ttl")
	}
}

func TestRedisReadCache_DeleteByPrefix(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	cache := NewRedisReadCache(client, "")
	ctx := context.Background()

	_ = cache.SetDishes(ctx, "product:dish:cid=1:status=1:name=:client=user", []domain.DishVO{{ID: 1}}, time.Minute)
	_ = cache.SetDishes(ctx, "product:dish:cid=2:status=1:name=:client=user", []domain.DishVO{{ID: 2}}, time.Minute)
	if err := cache.deleteByPrefix("product:dish:cid=1:"); err != nil {
		t.Fatalf("deleteByPrefix failed: %v", err)
	}

	_, ok1, _ := cache.GetDishes(ctx, "product:dish:cid=1:status=1:name=:client=user")
	_, ok2, _ := cache.GetDishes(ctx, "product:dish:cid=2:status=1:name=:client=user")
	if ok1 {
		t.Fatal("expected key with prefix to be deleted")
	}
	if !ok2 {
		t.Fatal("expected other key to remain")
	}
}
