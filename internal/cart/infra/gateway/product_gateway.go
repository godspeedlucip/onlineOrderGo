package gateway

import (
	"context"

	"go-baseline-skeleton/internal/cart/domain"
	productdomain "go-baseline-skeleton/internal/product/domain"
)

type productReadRepository interface {
	GetDishByID(ctx context.Context, id int64) (*productdomain.Dish, error)
	GetSetmealByID(ctx context.Context, id int64) (*productdomain.Setmeal, error)
}

type ProductGateway struct {
	readRepo productReadRepository
}

func NewProductGateway(readRepo productReadRepository) *ProductGateway {
	return &ProductGateway{readRepo: readRepo}
}

func (g *ProductGateway) GetDishSnapshot(ctx context.Context, dishID int64) (*domain.ItemSnapshot, error) {
	if g == nil || g.readRepo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read repository is not initialized", nil)
	}
	dish, err := g.readRepo.GetDishByID(ctx, dishID)
	if err != nil {
		return nil, err
	}
	if dish == nil {
		return nil, domain.NewBizError(domain.CodeNotFound, "dish not found", nil)
	}
	if dish.Status != productdomain.StatusEnabled {
		return nil, domain.NewBizError(domain.CodeConflict, "dish is disabled", nil)
	}
	return &domain.ItemSnapshot{
		ItemID:      dish.ID,
		Name:        dish.Name,
		Image:       dish.Image,
		Price:       dish.Price,
		SaleEnabled: true,
	}, nil
}

func (g *ProductGateway) GetSetmealSnapshot(ctx context.Context, setmealID int64) (*domain.ItemSnapshot, error) {
	if g == nil || g.readRepo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "product read repository is not initialized", nil)
	}
	setmeal, err := g.readRepo.GetSetmealByID(ctx, setmealID)
	if err != nil {
		return nil, err
	}
	if setmeal == nil {
		return nil, domain.NewBizError(domain.CodeNotFound, "setmeal not found", nil)
	}
	if setmeal.Status != productdomain.StatusEnabled {
		return nil, domain.NewBizError(domain.CodeConflict, "setmeal is disabled", nil)
	}
	return &domain.ItemSnapshot{
		ItemID:      setmeal.ID,
		Name:        setmeal.Name,
		Image:       setmeal.Image,
		Price:       setmeal.Price,
		SaleEnabled: true,
	}, nil
}
