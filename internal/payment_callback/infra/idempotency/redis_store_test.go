package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_AcquireAndDone(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "paycb:test")
	ctx := context.Background()

	token, ok, err := store.Acquire(ctx, "payment_callback", "notify-1", time.Minute)
	if err != nil || !ok || token == "" {
		t.Fatalf("Acquire failed token=%s ok=%v err=%v", token, ok, err)
	}
	_, ok, err = store.Acquire(ctx, "payment_callback", "notify-1", time.Minute)
	if err != nil || ok {
		t.Fatalf("duplicate Acquire expected false ok=%v err=%v", ok, err)
	}
	if err := store.MarkDone(ctx, "payment_callback", "notify-1", token); err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}
}
