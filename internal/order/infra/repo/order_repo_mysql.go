package repo

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"go-baseline-skeleton/internal/order/domain"
	ordertx "go-baseline-skeleton/internal/order/infra/tx"
)

type WriteTableRouter interface {
	ResolveWriteTable(ctx context.Context, at time.Time) (string, error)
}

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type MySQLOrderRepo struct {
	db      *sql.DB
	router  WriteTableRouter
	seq     atomic.Int64
}

func NewMySQLOrderRepo(db *sql.DB, router WriteTableRouter) *MySQLOrderRepo {
	return &MySQLOrderRepo{db: db, router: router}
}

func (r *MySQLOrderRepo) NextOrderNo(ctx context.Context) (string, error) {
	_ = ctx
	now := time.Now()
	seq := r.seq.Add(1) % 100000
	return fmt.Sprintf("O%s%05d", now.Format("20060102150405"), seq), nil
}

func (r *MySQLOrderRepo) SaveOrder(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	if order == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "order is nil", nil)
	}
	tableName, err := r.resolveWriteTable(ctx, order.CreatedAt)
	if err != nil {
		return err
	}
	exec := r.execer(ctx)
	res, err := exec.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s(number, user_id, status, amount, remark, version, order_time, update_time) VALUES(?, ?, ?, ?, ?, ?, ?, ?)", tableName),
		order.OrderNo, order.UserID, toDBStatus(order.Status), order.TotalAmount, order.Remark, order.Version, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	order.OrderID = id

	_, err = exec.ExecContext(ctx,
		"INSERT INTO order_table_index(order_id, table_name, order_no, created_at) VALUES(?, ?, ?, ?) ON DUPLICATE KEY UPDATE table_name=VALUES(table_name), order_no=VALUES(order_no)",
		order.OrderID, tableName, order.OrderNo, order.CreatedAt,
	)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = exec.ExecContext(ctx,
			"INSERT INTO order_detail(order_id, item_type, sku_id, name, flavor, quantity, unit_amount, line_amount, create_time) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			order.OrderID, strings.TrimSpace(item.ItemType), item.SkuID, item.Name, item.Flavor, item.Quantity, item.UnitAmount, item.LineAmount, order.CreatedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLOrderRepo) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	tableName, err := r.findTableByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if tableName == "" {
		tableName = "orders"
	}
	var (
		out       domain.Order
		statusRaw string
	)
	rowErr := r.execer(ctx).QueryRowContext(ctx,
		fmt.Sprintf("SELECT id, number, user_id, CAST(status AS CHAR), amount, remark, version, order_time, update_time FROM %s WHERE id=? LIMIT 1", tableName),
		orderID,
	).Scan(&out.OrderID, &out.OrderNo, &out.UserID, &statusRaw, &out.TotalAmount, &out.Remark, &out.Version, &out.CreatedAt, &out.UpdatedAt)
	if rowErr == sql.ErrNoRows {
		return nil, nil
	}
	if rowErr != nil {
		return nil, rowErr
	}
	out.Status = fromDBStatus(statusRaw)
	return &out, nil
}

func (r *MySQLOrderRepo) UpdateWithVersion(ctx context.Context, order *domain.Order, expectVersion int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	if order == nil || order.OrderID <= 0 {
		return false, domain.NewBizError(domain.CodeInvalidArgument, "invalid order", nil)
	}
	tableName, err := r.findTableByOrderID(ctx, order.OrderID)
	if err != nil {
		return false, err
	}
	if tableName == "" {
		tableName = "orders"
	}
	res, err := r.execer(ctx).ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET status=?, amount=?, remark=?, version=?, update_time=? WHERE id=? AND version=?", tableName),
		toDBStatus(order.Status), order.TotalAmount, order.Remark, order.Version, order.UpdatedAt, order.OrderID, expectVersion,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *MySQLOrderRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "order db is not initialized", nil)
	}
	return nil
}

func (r *MySQLOrderRepo) execer(ctx context.Context) sqlExecQuerier {
	if tx, ok := ordertx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}

func (r *MySQLOrderRepo) resolveWriteTable(ctx context.Context, at time.Time) (string, error) {
	if r.router == nil {
		return "orders", nil
	}
	tableName, err := r.router.ResolveWriteTable(ctx, at)
	if err != nil {
		return "", err
	}
	if !validTableName(tableName) {
		return "", domain.NewBizError(domain.CodeInvalidArgument, "invalid routed table name", nil)
	}
	return tableName, nil
}

func (r *MySQLOrderRepo) findTableByOrderID(ctx context.Context, orderID int64) (string, error) {
	var tableName string
	err := r.execer(ctx).QueryRowContext(ctx, "SELECT table_name FROM order_table_index WHERE order_id=? LIMIT 1", orderID).Scan(&tableName)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !validTableName(tableName) {
		return "", domain.NewBizError(domain.CodeInvalidArgument, "invalid table name in index", nil)
	}
	return tableName, nil
}

func toDBStatus(status domain.OrderStatus) int {
	switch status {
	case domain.OrderStatusPendingPay:
		return 1
	case domain.OrderStatusPaid:
		return 2
	case domain.OrderStatusAccepted:
		return 3
	case domain.OrderStatusDelivering:
		return 4
	case domain.OrderStatusCompleted:
		return 5
	case domain.OrderStatusCanceled:
		return 6
	default:
		return 0
	}
}

func fromDBStatus(raw string) domain.OrderStatus {
	v := strings.TrimSpace(raw)
	switch v {
	case "1", "PENDING_PAYMENT":
		return domain.OrderStatusPendingPay
	case "2", "PAID":
		return domain.OrderStatusPaid
	case "3", "ACCEPTED":
		return domain.OrderStatusAccepted
	case "4", "DELIVERING":
		return domain.OrderStatusDelivering
	case "5", "COMPLETED":
		return domain.OrderStatusCompleted
	case "6", "CANCELED", "CANCELLED":
		return domain.OrderStatusCanceled
	default:
		return domain.OrderStatus(v)
	}
}

var tableNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validTableName(tableName string) bool {
	if strings.TrimSpace(tableName) == "" {
		return false
	}
	return tableNamePattern.MatchString(tableName)
}

