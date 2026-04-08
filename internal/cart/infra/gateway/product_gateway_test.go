package gateway

import (
	"context"
	"testing"

	productdomain "go-baseline-skeleton/internal/product/domain"
)

type fakeReadRepo struct {
	dish    *productdomain.Dish
	setmeal *productdomain.Setmeal
}

func (f *fakeReadRepo) GetDishByID(ctx context.Context, id int64) (*productdomain.Dish, error) {
	_ = ctx
	return f.dish, nil
}

func (f *fakeReadRepo) GetSetmealByID(ctx context.Context, id int64) (*productdomain.Setmeal, error) {
	_ = ctx
	return f.setmeal, nil
}

func TestProductGateway_DishDisabledRejected(t *testing.T) {
	gw := NewProductGateway(&fakeReadRepo{
		dish: &productdomain.Dish{ID: 101, Name: "dish", Status: productdomain.StatusDisabled, Price: 100},
	})
	_, err := gw.GetDishSnapshot(context.Background(), 101)
	if err == nil {
		t.Fatal("expected disabled dish error")
	}
}

func TestProductGateway_SetmealEnabledSnapshot(t *testing.T) {
	gw := NewProductGateway(&fakeReadRepo{
		setmeal: &productdomain.Setmeal{ID: 201, Name: "setmeal", Status: productdomain.StatusEnabled, Price: 500},
	})
	snap, err := gw.GetSetmealSnapshot(context.Background(), 201)
	if err != nil {
		t.Fatalf("GetSetmealSnapshot failed: %v", err)
	}
	if snap == nil || !snap.SaleEnabled || snap.ItemID != 201 || snap.Price != 500 {
		t.Fatalf("snapshot mismatch: %+v", snap)
	}
}
