package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"go-baseline-skeleton/internal/baseline/domain"
	"go-baseline-skeleton/internal/baseline/infra/idempotency"
	"go-baseline-skeleton/internal/baseline/infra/tx"
)

type noopLogger struct{}

func (l *noopLogger) Info(ctx context.Context, msg string, fields map[string]any) {}
func (l *noopLogger) Error(ctx context.Context, msg string, err error, fields map[string]any) {}

func TestExecuteIdempotent_ReplaySucceededPayload(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	defer mini.Close()

	client := idempotency.NewRedisClient(mini.Addr(), "", 0)
	defer client.Close()

	store := idempotency.NewRedisStore(client, "baseline:test")
	usecase := NewBootstrapUsecase(
		tx.NewNoopManager(),
		&noopLogger{},
		&domain.Config{Idempotency: domain.IdempotencyConfig{Enabled: true, TTLSecond: 60}},
		nil,
		nil,
		nil,
		nil,
		nil,
		store,
	)

	var callCount int32
	action := func(ctx context.Context) (map[string]any, error) {
		atomic.AddInt32(&callCount, 1)
		return map[string]any{"orderId": "O-100"}, nil
	}

	first, err := usecase.ExecuteIdempotent(context.Background(), "order_create", "k-1", action)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if first.FromIdempotent {
		t.Fatal("first request should not be idempotent replay")
	}

	second, err := usecase.ExecuteIdempotent(context.Background(), "order_create", "k-1", action)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if !second.FromIdempotent {
		t.Fatal("second request should replay cached result")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("action should execute once, got %d", callCount)
	}
}

func TestExecuteIdempotent_ConcurrentConflict(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	defer mini.Close()

	client := idempotency.NewRedisClient(mini.Addr(), "", 0)
	defer client.Close()

	store := idempotency.NewRedisStore(client, "baseline:test")
	usecase := NewBootstrapUsecase(
		tx.NewNoopManager(),
		&noopLogger{},
		&domain.Config{Idempotency: domain.IdempotencyConfig{Enabled: true, TTLSecond: 60}},
		nil,
		nil,
		nil,
		nil,
		nil,
		store,
	)

	started := make(chan struct{})
	release := make(chan struct{})
	action := func(ctx context.Context) (map[string]any, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
		return map[string]any{"ok": true}, nil
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var firstErr error
	go func() {
		defer wg.Done()
		_, firstErr = usecase.ExecuteIdempotent(context.Background(), "order_create", "k-2", action)
	}()

	<-started

	_, secondErr := usecase.ExecuteIdempotent(context.Background(), "order_create", "k-2", func(ctx context.Context) (map[string]any, error) {
		return map[string]any{"should": "not-run"}, nil
	})
	close(release)
	wg.Wait()

	if firstErr != nil {
		t.Fatalf("first request unexpected error: %v", firstErr)
	}

	var bizErr *domain.BizError
	if !errors.As(secondErr, &bizErr) {
		t.Fatalf("second request should return BizError, got: %v", secondErr)
	}
	if bizErr.Code != domain.CodeConflict {
		t.Fatalf("unexpected error code: %s", bizErr.Code)
	}
}
