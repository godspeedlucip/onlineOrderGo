package app

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"go-baseline-skeleton/internal/cart/domain"
	cartidem "go-baseline-skeleton/internal/cart/infra/idempotency"
)

type memCartRepo struct {
	mu     sync.Mutex
	nextID int64
	items  map[string]domain.CartItem
}

func newMemCartRepo() *memCartRepo {
	return &memCartRepo{nextID: 1, items: make(map[string]domain.CartItem)}
}

func (r *memCartRepo) key(userID int64, k domain.CartItemKey) string {
	return string(k.ItemType) + "|" + strconv.FormatInt(userID, 10) + "|" + strconv.FormatInt(k.ItemID, 10) + "|" + k.Flavor
}

func (r *memCartRepo) GetByKey(ctx context.Context, userID int64, k domain.CartItemKey) (*domain.CartItem, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[r.key(userID, k)]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}

func (r *memCartRepo) Create(ctx context.Context, item domain.CartItem) (int64, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	k := domain.CartItemKey{ItemType: item.ItemType, ItemID: item.ItemID, Flavor: item.Flavor}
	key := r.key(item.UserID, k)
	if _, ok := r.items[key]; ok {
		return 0, domain.NewBizError(domain.CodeConflict, "cart unique key conflict", nil)
	}
	item.ID = r.nextID
	r.nextID++
	item.Version = 1
	item.Amount = item.UnitPrice * int64(item.Quantity)
	r.items[key] = item
	return item.ID, nil
}

func (r *memCartRepo) UpdateQuantity(ctx context.Context, id int64, quantity int, expectedVersion int64) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.items {
		if item.ID != id {
			continue
		}
		if expectedVersion > 0 && item.Version != expectedVersion {
			return false, nil
		}
		item.Quantity = quantity
		item.Amount = item.UnitPrice * int64(quantity)
		item.Version++
		r.items[key] = item
		return true, nil
	}
	return false, nil
}

func (r *memCartRepo) DeleteByID(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.items {
		if item.ID == id {
			delete(r.items, key)
			return true, nil
		}
	}
	return false, nil
}

func (r *memCartRepo) ListByUser(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.CartItem, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *memCartRepo) ClearByUser(ctx context.Context, userID int64) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.items {
		if item.UserID == userID {
			delete(r.items, key)
		}
	}
	return nil
}

type fakeProductGateway struct{}

func (g *fakeProductGateway) GetDishSnapshot(ctx context.Context, dishID int64) (*domain.ItemSnapshot, error) {
	_ = ctx
	return &domain.ItemSnapshot{ItemID: dishID, Name: "dish", Image: "img", Price: 100, SaleEnabled: true}, nil
}
func (g *fakeProductGateway) GetSetmealSnapshot(ctx context.Context, setmealID int64) (*domain.ItemSnapshot, error) {
	_ = ctx
	return &domain.ItemSnapshot{ItemID: setmealID, Name: "setmeal", Image: "img", Price: 200, SaleEnabled: true}, nil
}

type fixedUser struct{}

func (u *fixedUser) CurrentUserID(ctx context.Context) (int64, bool) { return 1, true }

type noopTx struct{}

func (t *noopTx) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

func TestService_ConcurrentAddItem(t *testing.T) {
	repo := newMemCartRepo()
	svc := NewService(Deps{
		Repo:     repo,
		Products: &fakeProductGateway{},
		Users:    &fixedUser{},
		Tx:       &noopTx{},
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.AddItem(context.Background(), domain.AddCartItemCmd{
				ItemType: domain.ItemTypeDish,
				ItemID:   101,
				Flavor:   "hot",
				Count:    1,
			}, "")
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent add failed: %v", err)
		}
	}

	items, err := svc.ListItems(context.Background())
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 20 {
		t.Fatalf("unexpected cart state: %+v", items)
	}
}

func TestService_ConcurrentSubItem(t *testing.T) {
	repo := newMemCartRepo()
	svc := NewService(Deps{
		Repo:     repo,
		Products: &fakeProductGateway{},
		Users:    &fixedUser{},
		Tx:       &noopTx{},
	})
	_, _ = svc.AddItem(context.Background(), domain.AddCartItemCmd{
		ItemType: domain.ItemTypeDish,
		ItemID:   101,
		Flavor:   "hot",
		Count:    100,
	}, "")

	var wg sync.WaitGroup
	errCh := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.SubItem(context.Background(), domain.SubCartItemCmd{
				ItemType: domain.ItemTypeDish,
				ItemID:   101,
				Flavor:   "hot",
				Count:    1,
			}, "")
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent sub failed: %v", err)
		}
	}
	items, err := svc.ListItems(context.Background())
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 70 {
		t.Fatalf("unexpected cart state: %+v", items)
	}
}

func TestService_DuplicateReplay(t *testing.T) {
	repo := newMemCartRepo()
	svc := NewService(Deps{
		Repo:        repo,
		Products:    &fakeProductGateway{},
		Users:       &fixedUser{},
		Tx:          &noopTx{},
		Idempotency: cartidem.NewInMemoryStore(),
	})
	first, err := svc.AddItem(context.Background(), domain.AddCartItemCmd{
		ItemType: domain.ItemTypeDish,
		ItemID:   101,
		Flavor:   "hot",
		Count:    1,
	}, "same-idem-key")
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	second, err := svc.AddItem(context.Background(), domain.AddCartItemCmd{
		ItemType: domain.ItemTypeDish,
		ItemID:   101,
		Flavor:   "hot",
		Count:    1,
	}, "same-idem-key")
	if err != nil {
		t.Fatalf("second add replay failed: %v", err)
	}
	if first.Quantity != second.Quantity || first.ID != second.ID {
		t.Fatalf("replay mismatch first=%+v second=%+v", first, second)
	}
	items, _ := svc.ListItems(context.Background())
	if len(items) != 1 || items[0].Quantity != 1 {
		t.Fatalf("duplicate replay should not re-execute write: %+v", items)
	}
}

type alwaysConflictRepo struct{}

func (r *alwaysConflictRepo) GetByKey(ctx context.Context, userID int64, k domain.CartItemKey) (*domain.CartItem, error) {
	return &domain.CartItem{ID: 1, UserID: userID, ItemType: k.ItemType, ItemID: k.ItemID, Flavor: k.Flavor, UnitPrice: 100, Quantity: 2, Version: 1}, nil
}
func (r *alwaysConflictRepo) Create(ctx context.Context, item domain.CartItem) (int64, error) { return 1, nil }
func (r *alwaysConflictRepo) UpdateQuantity(ctx context.Context, id int64, quantity int, expectedVersion int64) (bool, error) {
	return false, nil
}
func (r *alwaysConflictRepo) DeleteByID(ctx context.Context, id int64) (bool, error) { return false, nil }
func (r *alwaysConflictRepo) ListByUser(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	return nil, nil
}
func (r *alwaysConflictRepo) ClearByUser(ctx context.Context, userID int64) error { return nil }

func TestService_CASConflict(t *testing.T) {
	svc := NewService(Deps{
		Repo:     &alwaysConflictRepo{},
		Products: &fakeProductGateway{},
		Users:    &fixedUser{},
		Tx:       &noopTx{},
	})
	_, err := svc.UpdateQuantity(context.Background(), domain.UpdateCartQtyCmd{
		ItemType: domain.ItemTypeDish,
		ItemID:   101,
		Flavor:   "hot",
		Count:    3,
	}, "")
	if err == nil {
		t.Fatal("expected cas conflict error")
	}
	bizErr, ok := err.(*domain.BizError)
	if !ok || bizErr.Code != domain.CodeConflict {
		t.Fatalf("expected conflict biz error, got=%v", err)
	}
}
