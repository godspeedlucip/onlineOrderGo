package gateway

import (
	"context"
	"sync"

	"go-baseline-skeleton/internal/cart/domain"
)

type ProductGateway struct {
	mu       sync.RWMutex
	dishes   map[int64]domain.ItemSnapshot
	setmeals map[int64]domain.ItemSnapshot
}

func NewProductGateway() *ProductGateway {
	return &ProductGateway{
		dishes: map[int64]domain.ItemSnapshot{
			101: {ItemID: 101, Name: "Kung Pao Chicken", Image: "", Price: 3800, SaleEnabled: true},
			102: {ItemID: 102, Name: "Mapo Tofu", Image: "", Price: 2200, SaleEnabled: true},
		},
		setmeals: map[int64]domain.ItemSnapshot{
			201: {ItemID: 201, Name: "Lunch Combo", Image: "", Price: 5200, SaleEnabled: true},
		},
	}
}

func (g *ProductGateway) GetDishSnapshot(ctx context.Context, dishID int64) (*domain.ItemSnapshot, error) {
	_ = ctx
	g.mu.RLock()
	defer g.mu.RUnlock()
	snap, ok := g.dishes[dishID]
	if !ok {
		return nil, domain.NewBizError(domain.CodeNotFound, "dish not found", nil)
	}
	copy := snap
	return &copy, nil
}

func (g *ProductGateway) GetSetmealSnapshot(ctx context.Context, setmealID int64) (*domain.ItemSnapshot, error) {
	_ = ctx
	g.mu.RLock()
	defer g.mu.RUnlock()
	snap, ok := g.setmeals[setmealID]
	if !ok {
		return nil, domain.NewBizError(domain.CodeNotFound, "setmeal not found", nil)
	}
	copy := snap
	return &copy, nil
}

// TODO: wire to product module repository and align sale-status behavior with Java SQL.