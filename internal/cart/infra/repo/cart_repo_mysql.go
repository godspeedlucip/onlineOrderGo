package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"go-baseline-skeleton/internal/cart/domain"
)

type MySQLCartRepo struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]domain.CartItem
}

func NewMySQLCartRepo() *MySQLCartRepo {
	return &MySQLCartRepo{nextID: 1, items: make(map[int64]domain.CartItem)}
}

func (r *MySQLCartRepo) GetByKey(ctx context.Context, userID int64, key domain.CartItemKey) (*domain.CartItem, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if item.UserID == userID && item.ItemType == key.ItemType && item.ItemID == key.ItemID && item.Flavor == key.Flavor {
			copy := item
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *MySQLCartRepo) Create(ctx context.Context, item domain.CartItem) (int64, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	now := time.Now()
	item.ID = id
	item.Version = 1
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Amount = item.UnitPrice * int64(item.Quantity)
	r.items[id] = item
	return id, nil
}

func (r *MySQLCartRepo) UpdateQuantity(ctx context.Context, id int64, quantity int, expectedVersion int64) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return false, nil
	}
	if expectedVersion > 0 && item.Version != expectedVersion {
		return false, nil
	}
	item.Quantity = quantity
	item.Amount = item.UnitPrice * int64(item.Quantity)
	item.Version++
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return true, nil
}

func (r *MySQLCartRepo) DeleteByID(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return false, nil
	}
	delete(r.items, id)
	return true, nil
}

func (r *MySQLCartRepo) ListByUser(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.CartItem, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (r *MySQLCartRepo) ClearByUser(ctx context.Context, userID int64) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, item := range r.items {
		if item.UserID == userID {
			delete(r.items, id)
		}
	}
	return nil
}