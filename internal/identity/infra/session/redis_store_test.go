package session

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"go-baseline-skeleton/internal/identity/domain"
)

func TestRedisStore_CompareAndIncreaseVersion(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "identity:test")
	ctx := context.Background()

	ver, err := store.EnsureVersion(ctx, domain.AccountTypeEmployee, 1)
	if err != nil || ver != 1 {
		t.Fatalf("EnsureVersion mismatch: ver=%d err=%v", ver, err)
	}

	next, updated, err := store.CompareAndIncreaseVersion(ctx, domain.AccountTypeEmployee, 1, 1)
	if err != nil || !updated || next != 2 {
		t.Fatalf("CompareAndIncreaseVersion mismatch: next=%d updated=%v err=%v", next, updated, err)
	}

	_, updated, err = store.CompareAndIncreaseVersion(ctx, domain.AccountTypeEmployee, 1, 1)
	if err != nil || updated {
		t.Fatalf("expected CAS conflict, updated=%v err=%v", updated, err)
	}
}

func TestRedisStore_RevokeTokenTTL(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	store := NewRedisStore(client, "identity:test")
	ctx := context.Background()

	expireAt := time.Now().Add(2 * time.Second)
	if err := store.MarkTokenRevoked(ctx, "tid-1", expireAt); err != nil {
		t.Fatalf("MarkTokenRevoked failed: %v", err)
	}

	revoked, err := store.IsTokenRevoked(ctx, "tid-1")
	if err != nil || !revoked {
		t.Fatalf("expected token revoked, revoked=%v err=%v", revoked, err)
	}
}
