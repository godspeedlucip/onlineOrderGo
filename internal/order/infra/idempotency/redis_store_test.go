package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_AcquireAndReplayResult(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client, "order:idem:test")
	ctx := context.Background()
	token, ok, err := store.Acquire(ctx, "order:create", "k1", time.Minute)
	if err != nil || !ok {
		t.Fatalf("acquire failed ok=%v err=%v", ok, err)
	}
	if err := store.MarkDone(ctx, "order:create", "k1", token, []byte(`{"orderId":1}`)); err != nil {
		t.Fatalf("mark done failed: %v", err)
	}
	_, ok, err = store.Acquire(ctx, "order:create", "k1", time.Minute)
	if err != nil || ok {
		t.Fatalf("second acquire should be blocked ok=%v err=%v", ok, err)
	}
	payload, found, err := store.GetDoneResult(ctx, "order:create", "k1")
	if err != nil || !found || string(payload) != `{"orderId":1}` {
		t.Fatalf("unexpected replay result found=%v payload=%s err=%v", found, string(payload), err)
	}
}
