package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_ReplayDoneResult(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "cart:test:idem")
	ctx := context.Background()
	token, acquired, err := store.Acquire(ctx, "cart:add", "idemp-1", 30*time.Second)
	if err != nil || !acquired {
		t.Fatalf("Acquire failed token=%s acquired=%v err=%v", token, acquired, err)
	}
	if err := store.MarkDone(ctx, "cart:add", "idemp-1", token, []byte(`{"id":1,"quantity":3}`)); err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}
	_, acquired, err = store.Acquire(ctx, "cart:add", "idemp-1", 30*time.Second)
	if err != nil {
		t.Fatalf("second Acquire failed: %v", err)
	}
	if acquired {
		t.Fatal("duplicate key should not acquire")
	}
	raw, found, err := store.GetDoneResult(ctx, "cart:add", "idemp-1")
	if err != nil || !found {
		t.Fatalf("GetDoneResult failed found=%v err=%v", found, err)
	}
	if string(raw) != `{"id":1,"quantity":3}` {
		t.Fatalf("unexpected replay payload: %s", string(raw))
	}
}
