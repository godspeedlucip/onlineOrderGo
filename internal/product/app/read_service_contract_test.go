package app

import (
	"context"
	"testing"

	"go-baseline-skeleton/internal/product/domain"
)

type spyReadRepo struct {
	lastCategoryQuery domain.CategoryQuery
	lastDishQuery     domain.DishQuery
	lastSetmealQuery  domain.SetmealQuery
}

func (s *spyReadRepo) ListCategories(ctx context.Context, q domain.CategoryQuery) ([]domain.Category, error) {
	s.lastCategoryQuery = q
	return []domain.Category{}, nil
}

func (s *spyReadRepo) ListDishes(ctx context.Context, q domain.DishQuery) ([]domain.Dish, error) {
	s.lastDishQuery = q
	return []domain.Dish{}, nil
}

func (s *spyReadRepo) ListDishFlavorsByDishIDs(ctx context.Context, dishIDs []int64) (map[int64][]domain.DishFlavor, error) {
	return map[int64][]domain.DishFlavor{}, nil
}

func (s *spyReadRepo) GetDishByID(ctx context.Context, id int64) (*domain.Dish, error) { return nil, nil }

func (s *spyReadRepo) ListSetmeals(ctx context.Context, q domain.SetmealQuery) ([]domain.Setmeal, error) {
	s.lastSetmealQuery = q
	return []domain.Setmeal{}, nil
}

func (s *spyReadRepo) ListSetmealDishesBySetmealIDs(ctx context.Context, setmealIDs []int64) (map[int64][]domain.SetmealDish, error) {
	return map[int64][]domain.SetmealDish{}, nil
}

func (s *spyReadRepo) GetSetmealByID(ctx context.Context, id int64) (*domain.Setmeal, error) { return nil, nil }

func TestReadService_DefaultFilter_UserClient(t *testing.T) {
	repo := &spyReadRepo{}
	svc := NewReadService(ReadDeps{Repo: repo})

	_, _ = svc.ListCategories(context.Background(), domain.CategoryQuery{ClientTag: "user"})
	if repo.lastCategoryQuery.Status == nil || *repo.lastCategoryQuery.Status != domain.StatusEnabled {
		t.Fatalf("category default status mismatch: %+v", repo.lastCategoryQuery.Status)
	}

	_, _ = svc.ListDishes(context.Background(), domain.DishQuery{ClientTag: "user"})
	if repo.lastDishQuery.Status == nil || *repo.lastDishQuery.Status != domain.StatusEnabled {
		t.Fatalf("dish default status mismatch: %+v", repo.lastDishQuery.Status)
	}

	_, _ = svc.ListSetmeals(context.Background(), domain.SetmealQuery{ClientTag: "user"})
	if repo.lastSetmealQuery.Status == nil || *repo.lastSetmealQuery.Status != domain.StatusEnabled {
		t.Fatalf("setmeal default status mismatch: %+v", repo.lastSetmealQuery.Status)
	}
}

func TestReadService_EmptyResultSemantics(t *testing.T) {
	repo := &spyReadRepo{}
	svc := NewReadService(ReadDeps{Repo: repo})

	cats, err := svc.ListCategories(context.Background(), domain.CategoryQuery{ClientTag: "user"})
	if err != nil || cats == nil || len(cats) != 0 {
		t.Fatalf("categories empty semantics mismatch: err=%v cats=%+v", err, cats)
	}

	dishes, err := svc.ListDishes(context.Background(), domain.DishQuery{ClientTag: "user"})
	if err != nil || dishes == nil || len(dishes) != 0 {
		t.Fatalf("dishes empty semantics mismatch: err=%v dishes=%+v", err, dishes)
	}

	setmeals, err := svc.ListSetmeals(context.Background(), domain.SetmealQuery{ClientTag: "user"})
	if err != nil || setmeals == nil || len(setmeals) != 0 {
		t.Fatalf("setmeals empty semantics mismatch: err=%v setmeals=%+v", err, setmeals)
	}
}
