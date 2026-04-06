package repo

import (
	"context"
	"sort"
	"strings"

	"go-baseline-skeleton/internal/product/domain"
)

// MySQLReadRepository currently provides an in-memory runnable baseline.
// TODO: replace each method body with SQL implementation while preserving output behavior.
type MySQLReadRepository struct {
	store *memoryStore
}

func NewMySQLReadRepository() *MySQLReadRepository {
	return &MySQLReadRepository{store: defaultStore}
}

func (r *MySQLReadRepository) ListCategories(ctx context.Context, q domain.CategoryQuery) ([]domain.Category, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	out := make([]domain.Category, 0, len(r.store.categories))
	for _, item := range r.store.categories {
		if q.Type != nil && item.Type != *q.Type {
			continue
		}
		if q.Status != nil && item.Status != *q.Status {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort == out[j].Sort {
			return out[i].UpdateTime.After(out[j].UpdateTime)
		}
		return out[i].Sort < out[j].Sort
	})
	return out, nil
}

func (r *MySQLReadRepository) ListDishes(ctx context.Context, q domain.DishQuery) ([]domain.Dish, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	kw := strings.TrimSpace(strings.ToLower(q.Name))
	out := make([]domain.Dish, 0, len(r.store.dishes))
	for _, item := range r.store.dishes {
		if q.CategoryID != nil && item.CategoryID != *q.CategoryID {
			continue
		}
		if q.Status != nil && item.Status != *q.Status {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(item.Name), kw) {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort == out[j].Sort {
			return out[i].UpdateTime.After(out[j].UpdateTime)
		}
		return out[i].Sort < out[j].Sort
	})
	return out, nil
}

func (r *MySQLReadRepository) ListDishFlavorsByDishIDs(ctx context.Context, dishIDs []int64) (map[int64][]domain.DishFlavor, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	set := make(map[int64]struct{}, len(dishIDs))
	for _, id := range dishIDs {
		set[id] = struct{}{}
	}
	out := make(map[int64][]domain.DishFlavor)
	for dishID, flavors := range r.store.dishFlavors {
		if _, ok := set[dishID]; !ok {
			continue
		}
		copyFlavors := make([]domain.DishFlavor, len(flavors))
		copy(copyFlavors, flavors)
		out[dishID] = copyFlavors
	}
	return out, nil
}

func (r *MySQLReadRepository) GetDishByID(ctx context.Context, id int64) (*domain.Dish, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	item, ok := r.store.dishes[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (r *MySQLReadRepository) ListSetmeals(ctx context.Context, q domain.SetmealQuery) ([]domain.Setmeal, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	kw := strings.TrimSpace(strings.ToLower(q.Name))
	out := make([]domain.Setmeal, 0, len(r.store.setmeals))
	for _, item := range r.store.setmeals {
		if q.CategoryID != nil && item.CategoryID != *q.CategoryID {
			continue
		}
		if q.Status != nil && item.Status != *q.Status {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(item.Name), kw) {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdateTime.Equal(out[j].UpdateTime) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdateTime.After(out[j].UpdateTime)
	})
	return out, nil
}

func (r *MySQLReadRepository) ListSetmealDishesBySetmealIDs(ctx context.Context, setmealIDs []int64) (map[int64][]domain.SetmealDish, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	set := make(map[int64]struct{}, len(setmealIDs))
	for _, id := range setmealIDs {
		set[id] = struct{}{}
	}
	out := make(map[int64][]domain.SetmealDish)
	for setmealID, items := range r.store.setmealDishes {
		if _, ok := set[setmealID]; !ok {
			continue
		}
		copyItems := make([]domain.SetmealDish, len(items))
		copy(copyItems, items)
		out[setmealID] = copyItems
	}
	return out, nil
}

func (r *MySQLReadRepository) GetSetmealByID(ctx context.Context, id int64) (*domain.Setmeal, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	item, ok := r.store.setmeals[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}