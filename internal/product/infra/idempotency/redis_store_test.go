package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_ReplayDoneSnapshot(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "product:test:idem")
	ctx := context.Background()

	token, acquired, err := store.Acquire(ctx, "sceneA", "keyA", 10*time.Second)
	if err != nil || !acquired {
		t.Fatalf("Acquire failed: token=%s acquired=%v err=%v", token, acquired, err)
	}
	if err := store.MarkDone(ctx, "sceneA", "keyA", token, []byte(`{"id":123}`)); err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}

	_, acquired, err = store.Acquire(ctx, "sceneA", "keyA", 10*time.Second)
	if err != nil {
		t.Fatalf("second Acquire failed: %v", err)
	}
	if acquired {
		t.Fatal("duplicate key should not acquire")
	}
	got, found, err := store.GetDoneResult(ctx, "sceneA", "keyA")
	if err != nil || !found {
		t.Fatalf("GetDoneResult failed: found=%v err=%v", found, err)
	}
	if string(got) != `{"id":123}` {
		t.Fatalf("snapshot mismatch: %s", string(got))
	}
}
