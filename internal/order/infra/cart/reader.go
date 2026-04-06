package cart

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

type Reader struct{}

func NewReader() *Reader { return &Reader{} }

func (r *Reader) LoadCheckedItems(ctx context.Context, userID int64) ([]domain.OrderItem, int64, error) {
	_ = ctx
	_ = userID
	// TODO: replace with real cart query; current data is for local runnable baseline only.
	items := []domain.OrderItem{{
		SkuID:      1,
		Name:       "demo-item",
		Quantity:   1,
		UnitAmount: 1000,
		LineAmount: 1000,
	}}
	return items, 1000, nil
}
