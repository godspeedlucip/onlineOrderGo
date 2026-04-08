package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"go-baseline-skeleton/internal/cart/domain"
	carttx "go-baseline-skeleton/internal/cart/infra/tx"
)

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type MySQLCartRepo struct {
	db *sql.DB
}

func NewMySQLCartRepo(db *sql.DB) *MySQLCartRepo {
	return &MySQLCartRepo{db: db}
}

func (r *MySQLCartRepo) GetByKey(ctx context.Context, userID int64, key domain.CartItemKey) (*domain.CartItem, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	var out domain.CartItem
	err := r.execer(ctx).QueryRowContext(ctx,
		"SELECT id, user_id, item_type, item_id, flavor, name, image, unit_price, quantity, amount, version, create_time, update_time "+
			"FROM cart WHERE user_id=? AND item_type=? AND item_id=? AND flavor=? LIMIT 1",
		userID, string(key.ItemType), key.ItemID, key.Flavor,
	).Scan(
		&out.ID, &out.UserID, &out.ItemType, &out.ItemID, &out.Flavor, &out.Name, &out.Image,
		&out.UnitPrice, &out.Quantity, &out.Amount, &out.Version, &out.CreatedAt, &out.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *MySQLCartRepo) Create(ctx context.Context, item domain.CartItem) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	now := time.Now()
	item.Flavor = strings.TrimSpace(item.Flavor)
	item.Amount = item.UnitPrice * int64(item.Quantity)
	res, err := r.execer(ctx).ExecContext(ctx,
		"INSERT INTO cart (user_id, item_type, item_id, flavor, name, image, unit_price, quantity, amount, version, create_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)",
		item.UserID, string(item.ItemType), item.ItemID, item.Flavor, item.Name, item.Image, item.UnitPrice, item.Quantity, item.Amount, now, now,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return 0, domain.NewBizError(domain.CodeConflict, "cart unique key conflict", err)
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *MySQLCartRepo) UpdateQuantity(ctx context.Context, id int64, quantity int, expectedVersion int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	if quantity < 0 {
		return false, domain.NewBizError(domain.CodeInvalidArgument, "quantity must be >= 0", nil)
	}
	query := "UPDATE cart SET quantity=?, amount=unit_price*?, version=version+1, update_time=? WHERE id=?"
	args := []any{quantity, quantity, time.Now(), id}
	if expectedVersion > 0 {
		query += " AND version=?"
		args = append(args, expectedVersion)
	}
	res, err := r.execer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *MySQLCartRepo) DeleteByID(ctx context.Context, id int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	res, err := r.execer(ctx).ExecContext(ctx, "DELETE FROM cart WHERE id=?", id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *MySQLCartRepo) ListByUser(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.execer(ctx).QueryContext(ctx,
		"SELECT id, user_id, item_type, item_id, flavor, name, image, unit_price, quantity, amount, version, create_time, update_time "+
			"FROM cart WHERE user_id=? ORDER BY update_time DESC, id DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CartItem, 0)
	for rows.Next() {
		var item domain.CartItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.ItemType, &item.ItemID, &item.Flavor, &item.Name, &item.Image,
			&item.UnitPrice, &item.Quantity, &item.Amount, &item.Version, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *MySQLCartRepo) ClearByUser(ctx context.Context, userID int64) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx, "DELETE FROM cart WHERE user_id=?", userID)
	return err
}

func (r *MySQLCartRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "cart db is not initialized", nil)
	}
	return nil
}

func (r *MySQLCartRepo) execer(ctx context.Context) sqlExecQuerier {
	if tx, ok := carttx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "unique constraint")
}
