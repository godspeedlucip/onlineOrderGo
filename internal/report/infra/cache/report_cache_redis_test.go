package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisReportCache_SetAndGet(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	cache := NewRedisReportCache(client, "report:test")
	ctx := context.Background()

	payload := map[string]any{"orderCount": 2, "turnover": 3000}
	if err := cache.Set(ctx, "overview:k1", payload, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var out map[string]any
	ok, err := cache.Get(ctx, "overview:k1", &out)
	if err != nil || !ok {
		t.Fatalf("Get failed ok=%v err=%v", ok, err)
	}
	if out["orderCount"].(float64) != 2 || out["turnover"].(float64) != 3000 {
		t.Fatalf("unexpected cached payload: %+v", out)
	}
}
