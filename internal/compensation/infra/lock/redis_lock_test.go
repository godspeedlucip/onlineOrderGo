package lock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisLocker_TryLockAndRelease(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	locker := NewRedisLocker(client, "compensation:lock:test")
	unlock, ok, err := locker.TryLock(context.Background(), "job:order", time.Minute)
	if err != nil || !ok {
		t.Fatalf("TryLock failed ok=%v err=%v", ok, err)
	}
	_, ok2, err := locker.TryLock(context.Background(), "job:order", time.Minute)
	if err != nil {
		t.Fatalf("second TryLock error: %v", err)
	}
	if ok2 {
		t.Fatal("expected second lock acquisition to fail")
	}
	if err := unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	_, ok3, err := locker.TryLock(context.Background(), "job:order", time.Minute)
	if err != nil || !ok3 {
		t.Fatalf("lock should be acquirable after unlock ok=%v err=%v", ok3, err)
	}
}
