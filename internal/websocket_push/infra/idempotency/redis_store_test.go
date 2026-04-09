package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_TryAcquire(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "ws")
	acquired, err := store.TryAcquire(context.Background(), "m1", time.Minute)
	if err != nil {
		t.Fatalf("try acquire first: %v", err)
	}
	if !acquired {
		t.Fatalf("expect first acquire=true")
	}
	acquired, err = store.TryAcquire(context.Background(), "m1", time.Minute)
	if err != nil {
		t.Fatalf("try acquire second: %v", err)
	}
	if acquired {
		t.Fatalf("expect second acquire=false")
	}
}
