package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type fakeWriteRepo struct {
	mu          sync.Mutex
	createCount int
}

func (f *fakeWriteRepo) CreateCategory(ctx context.Context, c domain.Category) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCount++
	return 11, nil
}
func (f *fakeWriteRepo) UpdateCategory(ctx context.Context, c domain.Category, expectedVersion int64) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) UpdateCategoryStatus(ctx context.Context, id int64, status int) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) DeleteCategory(ctx context.Context, id int64) (bool, error) { return true, nil }
func (f *fakeWriteRepo) CreateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor) (int64, error) {
	return 21, nil
}
func (f *fakeWriteRepo) UpdateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor, expectedVersion int64) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) UpdateDishStatus(ctx context.Context, id int64, status int) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) DeleteDish(ctx context.Context, id int64) (bool, error) { return true, nil }
func (f *fakeWriteRepo) CreateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish) (int64, error) {
	return 31, nil
}
func (f *fakeWriteRepo) UpdateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish, expectedVersion int64) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) UpdateSetmealStatus(ctx context.Context, id int64, status int) (bool, error) {
	return true, nil
}
func (f *fakeWriteRepo) DeleteSetmeal(ctx context.Context, id int64) (bool, error) { return true, nil }
func (f *fakeWriteRepo) ExistsDishUsedBySetmeal(ctx context.Context, dishID int64) (bool, error) {
	return false, nil
}
func (f *fakeWriteRepo) ExistsCategoryUsedByDish(ctx context.Context, categoryID int64) (bool, error) {
	return false, nil
}
func (f *fakeWriteRepo) ExistsCategoryUsedBySetmeal(ctx context.Context, categoryID int64) (bool, error) {
	return false, nil
}

type fakeTx struct{}

func (f *fakeTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fakeIdempotency struct {
	mu       sync.Mutex
	acquired bool
	snapshot []byte
}

func (f *fakeIdempotency) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.acquired {
		f.acquired = true
		return "tok-1", true, nil
	}
	return "tok-1", false, nil
}
func (f *fakeIdempotency) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshot = append([]byte(nil), result...)
	return nil
}
func (f *fakeIdempotency) MarkFailed(ctx context.Context, scene, key, token, reason string) error { return nil }
func (f *fakeIdempotency) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.snapshot) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), f.snapshot...), true, nil
}

type fakeInvalidator struct {
	err error
}

func (f *fakeInvalidator) InvalidateCategory(ctx context.Context, categoryID int64) error  { return f.err }
func (f *fakeInvalidator) InvalidateDish(ctx context.Context, dishID int64, categoryID int64) error {
	return f.err
}
func (f *fakeInvalidator) InvalidateSetmeal(ctx context.Context, setmealID int64, categoryID int64) error {
	return f.err
}
func (f *fakeInvalidator) InvalidateByCategory(ctx context.Context, categoryID int64) error { return f.err }

type fakeOutbox struct {
	mu       sync.Mutex
	enqueued []domain.CacheInvalidateTask
}

func (f *fakeOutbox) Enqueue(ctx context.Context, task domain.CacheInvalidateTask) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enqueued = append(f.enqueued, task)
	return nil
}
func (f *fakeOutbox) RunOnce(ctx context.Context, invalidator domain.ProductCacheInvalidator, limit int) (int, error) {
	return 0, nil
}

func TestWriteService_IdempotencyReplaySuccessResult(t *testing.T) {
	repo := &fakeWriteRepo{}
	idem := &fakeIdempotency{}
	svc := NewWriteService(WriteDeps{
		Repo:        repo,
		Tx:          &fakeTx{},
		Idempotency: idem,
	})

	id1, err := svc.CreateCategory(context.Background(), domain.CreateCategoryCmd{Name: "M", Type: 1, Sort: 1}, "same-key")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	id2, err := svc.CreateCategory(context.Background(), domain.CreateCategoryCmd{Name: "M", Type: 1, Sort: 1}, "same-key")
	if err != nil {
		t.Fatalf("replay create failed: %v", err)
	}
	if id1 != 11 || id2 != 11 {
		t.Fatalf("id mismatch id1=%d id2=%d", id1, id2)
	}
	if repo.createCount != 1 {
		t.Fatalf("repo should be called once, got=%d", repo.createCount)
	}
}

func TestWriteService_InvalidationFailEnqueueOutbox(t *testing.T) {
	repo := &fakeWriteRepo{}
	outbox := &fakeOutbox{}
	svc := NewWriteService(WriteDeps{
		Repo:        repo,
		Tx:          &fakeTx{},
		Invalidator: &fakeInvalidator{err: errors.New("redis unavailable")},
		Outbox:      outbox,
	})

	id, err := svc.CreateDish(context.Background(), domain.CreateDishCmd{
		CategoryID: 2,
		Name:       "Fish",
		Status:     1,
	}, "")
	if err != nil {
		t.Fatalf("CreateDish failed: %v", err)
	}
	if id != 21 {
		t.Fatalf("id mismatch: %d", id)
	}
	if len(outbox.enqueued) != 1 {
		t.Fatalf("expected one outbox task, got=%d", len(outbox.enqueued))
	}
	task := outbox.enqueued[0]
	if task.Operation != "dish" || task.CategoryID != 2 || task.EntityID != 21 {
		t.Fatalf("unexpected outbox task: %+v", task)
	}
	if task.EnqueueAtMS <= 0 {
		t.Fatalf("enqueue timestamp should be set: %+v", task)
	}
}
