package cart

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go-baseline-skeleton/internal/order/domain"
	ordertx "go-baseline-skeleton/internal/order/infra/tx"
)

type sqlQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Reader struct {
	db                   *sql.DB
	deliveryFee          int64
	couponDiscount       int64
	fullReductionTrigger int64
	fullReductionAmount  int64
}

func NewReader(db *sql.DB, deliveryFee, couponDiscount, fullReductionTrigger, fullReductionAmount int64) *Reader {
	return &Reader{
		db:                   db,
		deliveryFee:          deliveryFee,
		couponDiscount:       couponDiscount,
		fullReductionTrigger: fullReductionTrigger,
		fullReductionAmount:  fullReductionAmount,
	}
}

func (r *Reader) LoadCheckedItems(ctx context.Context, userID int64) ([]domain.OrderItem, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, domain.NewBizError(domain.CodeInternal, "cart db is not initialized", nil)
	}
	rows, err := r.queryer(ctx).QueryContext(ctx,
		"SELECT item_type, item_id, flavor, name, unit_price, quantity FROM cart WHERE user_id=? ORDER BY update_time DESC, id DESC",
		userID,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	subtotal := int64(0)
	for rows.Next() {
		var (
			itemType   string
			itemID     int64
			flavor     string
			name       string
			unitAmount int64
			quantity   int64
		)
		if scanErr := rows.Scan(&itemType, &itemID, &flavor, &name, &unitAmount, &quantity); scanErr != nil {
			return nil, 0, scanErr
		}
		if quantity <= 0 || unitAmount <= 0 {
			return nil, 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid cart item amount", nil)
		}
		if err := r.ensureItemSaleEnabled(ctx, itemType, itemID); err != nil {
			return nil, 0, err
		}
		lineAmount := unitAmount * quantity
		items = append(items, domain.OrderItem{
			ItemType:   normalizeItemType(itemType),
			SkuID:      itemID,
			Flavor:     strings.TrimSpace(flavor),
			Name:       name,
			Quantity:   quantity,
			UnitAmount: unitAmount,
			LineAmount: lineAmount,
		})
		subtotal += lineAmount
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, rowsErr
	}
	if len(items) == 0 {
		return []domain.OrderItem{}, 0, nil
	}
	discount := r.couponDiscount
	if r.fullReductionTrigger > 0 && subtotal >= r.fullReductionTrigger {
		discount += r.fullReductionAmount
	}
	total := subtotal + r.deliveryFee - discount
	if total <= 0 {
		return nil, 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid total amount after discount", nil)
	}
	return items, total, nil
}

func (r *Reader) ensureItemSaleEnabled(ctx context.Context, itemType string, itemID int64) error {
	t := normalizeItemType(itemType)
	var (
		status    int
		isDeleted int
	)
	switch t {
	case "dish":
		if err := r.queryer(ctx).QueryRowContext(ctx, "SELECT status, is_deleted FROM dish WHERE id=? LIMIT 1", itemID).Scan(&status, &isDeleted); err != nil {
			if err == sql.ErrNoRows {
				return domain.NewBizError(domain.CodeConflict, "dish not found", nil)
			}
			return err
		}
	case "setmeal":
		if err := r.queryer(ctx).QueryRowContext(ctx, "SELECT status, is_deleted FROM setmeal WHERE id=? LIMIT 1", itemID).Scan(&status, &isDeleted); err != nil {
			if err == sql.ErrNoRows {
				return domain.NewBizError(domain.CodeConflict, "setmeal not found", nil)
			}
			return err
		}
	default:
		return domain.NewBizError(domain.CodeInvalidArgument, fmt.Sprintf("unknown item type: %s", itemType), nil)
	}
	if isDeleted == 1 || status != 1 {
		return domain.NewBizError(domain.CodeConflict, "item is not sale-enabled", nil)
	}
	return nil
}

func normalizeItemType(itemType string) string {
	s := strings.ToLower(strings.TrimSpace(itemType))
	if s == "" {
		return "dish"
	}
	return s
}

func (r *Reader) queryer(ctx context.Context) sqlQueryer {
	if tx, ok := ordertx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}
