package repo

import (
	"context"
	"strings"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type MySQLWriteRepository struct {
	store *memoryStore
}

func NewMySQLWriteRepository() *MySQLWriteRepository {
	return &MySQLWriteRepository{store: defaultStore}
}

func (r *MySQLWriteRepository) CreateCategory(ctx context.Context, c domain.Category) (int64, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	for _, old := range r.store.categories {
		if old.Type == c.Type && strings.EqualFold(old.Name, c.Name) {
			return 0, domain.NewBizError(domain.CodeConflict, "category name exists", nil)
		}
	}

	id := r.store.nextCategoryID
	r.store.nextCategoryID++
	c.ID = id
	if c.Status != domain.StatusEnabled && c.Status != domain.StatusDisabled {
		c.Status = domain.StatusEnabled
	}
	c.UpdateTime = time.Now()
	r.store.categories[id] = c
	r.store.categoryVersion[id] = 1
	return id, nil
}

func (r *MySQLWriteRepository) UpdateCategory(ctx context.Context, c domain.Category, expectedVersion int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.categories[c.ID]
	if !ok {
		return false, nil
	}
	if expectedVersion > 0 && r.store.categoryVersion[c.ID] != expectedVersion {
		return false, nil
	}
	for id, item := range r.store.categories {
		if id == c.ID {
			continue
		}
		if item.Type == c.Type && strings.EqualFold(item.Name, c.Name) {
			return false, domain.NewBizError(domain.CodeConflict, "category name exists", nil)
		}
	}

	old.Name = c.Name
	old.Type = c.Type
	old.Sort = c.Sort
	old.UpdateTime = time.Now()
	r.store.categories[c.ID] = old
	r.store.categoryVersion[c.ID] = r.store.categoryVersion[c.ID] + 1
	return true, nil
}

func (r *MySQLWriteRepository) UpdateCategoryStatus(ctx context.Context, id int64, status int) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.categories[id]
	if !ok {
		return false, nil
	}
	old.Status = status
	old.UpdateTime = time.Now()
	r.store.categories[id] = old
	r.store.categoryVersion[id] = r.store.categoryVersion[id] + 1
	return true, nil
}

func (r *MySQLWriteRepository) DeleteCategory(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.categories[id]; !ok {
		return false, nil
	}
	delete(r.store.categories, id)
	delete(r.store.categoryVersion, id)
	return true, nil
}

func (r *MySQLWriteRepository) CreateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor) (int64, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	for _, old := range r.store.dishes {
		if old.CategoryID == d.CategoryID && strings.EqualFold(old.Name, d.Name) {
			return 0, domain.NewBizError(domain.CodeConflict, "dish name exists", nil)
		}
	}

	id := r.store.nextDishID
	r.store.nextDishID++
	d.ID = id
	d.UpdateTime = time.Now()
	r.store.dishes[id] = d
	r.store.dishVersion[id] = 1

	fls := make([]domain.DishFlavor, 0, len(flavors))
	for _, f := range flavors {
		f.ID = r.store.nextFlavorID
		r.store.nextFlavorID++
		f.DishID = id
		fls = append(fls, f)
	}
	r.store.dishFlavors[id] = fls
	return id, nil
}

func (r *MySQLWriteRepository) UpdateDishWithFlavors(ctx context.Context, d domain.Dish, flavors []domain.DishFlavor, expectedVersion int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.dishes[d.ID]
	if !ok {
		return false, nil
	}
	if expectedVersion > 0 && r.store.dishVersion[d.ID] != expectedVersion {
		return false, nil
	}
	for id, item := range r.store.dishes {
		if id == d.ID {
			continue
		}
		if item.CategoryID == d.CategoryID && strings.EqualFold(item.Name, d.Name) {
			return false, domain.NewBizError(domain.CodeConflict, "dish name exists", nil)
		}
	}

	old.CategoryID = d.CategoryID
	old.Name = d.Name
	old.Price = d.Price
	old.Image = d.Image
	old.Description = d.Description
	old.Status = d.Status
	old.UpdateTime = time.Now()
	r.store.dishes[d.ID] = old
	r.store.dishVersion[d.ID] = r.store.dishVersion[d.ID] + 1

	fls := make([]domain.DishFlavor, 0, len(flavors))
	for _, f := range flavors {
		f.ID = r.store.nextFlavorID
		r.store.nextFlavorID++
		f.DishID = d.ID
		fls = append(fls, f)
	}
	r.store.dishFlavors[d.ID] = fls
	return true, nil
}

func (r *MySQLWriteRepository) UpdateDishStatus(ctx context.Context, id int64, status int) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.dishes[id]
	if !ok {
		return false, nil
	}
	old.Status = status
	old.UpdateTime = time.Now()
	r.store.dishes[id] = old
	r.store.dishVersion[id] = r.store.dishVersion[id] + 1
	return true, nil
}

func (r *MySQLWriteRepository) DeleteDish(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.dishes[id]; !ok {
		return false, nil
	}
	delete(r.store.dishes, id)
	delete(r.store.dishVersion, id)
	delete(r.store.dishFlavors, id)
	return true, nil
}

func (r *MySQLWriteRepository) CreateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish) (int64, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	for _, old := range r.store.setmeals {
		if old.CategoryID == s.CategoryID && strings.EqualFold(old.Name, s.Name) {
			return 0, domain.NewBizError(domain.CodeConflict, "setmeal name exists", nil)
		}
	}

	id := r.store.nextSetmealID
	r.store.nextSetmealID++
	s.ID = id
	s.UpdateTime = time.Now()
	r.store.setmeals[id] = s
	r.store.setmealVersion[id] = 1

	in := make([]domain.SetmealDish, 0, len(items))
	for _, item := range items {
		item.ID = r.store.nextSetmealDishID
		r.store.nextSetmealDishID++
		item.SetmealID = id
		in = append(in, item)
	}
	r.store.setmealDishes[id] = in
	return id, nil
}

func (r *MySQLWriteRepository) UpdateSetmealWithItems(ctx context.Context, s domain.Setmeal, items []domain.SetmealDish, expectedVersion int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.setmeals[s.ID]
	if !ok {
		return false, nil
	}
	if expectedVersion > 0 && r.store.setmealVersion[s.ID] != expectedVersion {
		return false, nil
	}
	for id, item := range r.store.setmeals {
		if id == s.ID {
			continue
		}
		if item.CategoryID == s.CategoryID && strings.EqualFold(item.Name, s.Name) {
			return false, domain.NewBizError(domain.CodeConflict, "setmeal name exists", nil)
		}
	}

	old.CategoryID = s.CategoryID
	old.Name = s.Name
	old.Price = s.Price
	old.Image = s.Image
	old.Description = s.Description
	old.Status = s.Status
	old.UpdateTime = time.Now()
	r.store.setmeals[s.ID] = old
	r.store.setmealVersion[s.ID] = r.store.setmealVersion[s.ID] + 1

	in := make([]domain.SetmealDish, 0, len(items))
	for _, item := range items {
		item.ID = r.store.nextSetmealDishID
		r.store.nextSetmealDishID++
		item.SetmealID = s.ID
		in = append(in, item)
	}
	r.store.setmealDishes[s.ID] = in
	return true, nil
}

func (r *MySQLWriteRepository) UpdateSetmealStatus(ctx context.Context, id int64, status int) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	old, ok := r.store.setmeals[id]
	if !ok {
		return false, nil
	}
	old.Status = status
	old.UpdateTime = time.Now()
	r.store.setmeals[id] = old
	r.store.setmealVersion[id] = r.store.setmealVersion[id] + 1
	return true, nil
}

func (r *MySQLWriteRepository) DeleteSetmeal(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.setmeals[id]; !ok {
		return false, nil
	}
	delete(r.store.setmeals, id)
	delete(r.store.setmealVersion, id)
	delete(r.store.setmealDishes, id)
	return true, nil
}

func (r *MySQLWriteRepository) ExistsDishUsedBySetmeal(ctx context.Context, dishID int64) (bool, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, items := range r.store.setmealDishes {
		for _, item := range items {
			if item.DishID == dishID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *MySQLWriteRepository) ExistsCategoryUsedByDish(ctx context.Context, categoryID int64) (bool, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, dish := range r.store.dishes {
		if dish.CategoryID == categoryID {
			return true, nil
		}
	}
	return false, nil
}

func (r *MySQLWriteRepository) ExistsCategoryUsedBySetmeal(ctx context.Context, categoryID int64) (bool, error) {
	_ = ctx
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, setmeal := range r.store.setmeals {
		if setmeal.CategoryID == categoryID {
			return true, nil
		}
	}
	return false, nil
}