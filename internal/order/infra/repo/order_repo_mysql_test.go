package repo

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/order/domain"
)

type fakeWriteRouter struct{ table string }

func (r *fakeWriteRouter) ResolveWriteTable(ctx context.Context, at time.Time) (string, error) {
	_ = ctx
	_ = at
	return r.table, nil
}

func TestMySQLOrderRepo_SaveOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLOrderRepo(db, &fakeWriteRouter{table: "orders_202604"})
	now := time.Now()
	order := &domain.Order{OrderNo: "O202604090001", UserID: 1001, Status: domain.OrderStatusPendingPay, TotalAmount: 1200, Remark: "r", Version: 1, CreatedAt: now, UpdatedAt: now}
	items := []domain.OrderItem{{ItemType: "dish", SkuID: 1, Name: "A", Flavor: "", Quantity: 2, UnitAmount: 600, LineAmount: 1200}}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO orders_202604(number, user_id, status, amount, remark, version, order_time, update_time) VALUES(?, ?, ?, ?, ?, ?, ?, ?)"))
		.WithArgs(order.OrderNo, order.UserID, toDBStatus(order.Status), order.TotalAmount, order.Remark, order.Version, order.CreatedAt, order.UpdatedAt)
		.WillReturnResult(sqlmock.NewResult(101, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_table_index(order_id, table_name, order_no, created_at) VALUES(?, ?, ?, ?) ON DUPLICATE KEY UPDATE table_name=VALUES(table_name), order_no=VALUES(order_no)"))
		.WithArgs(int64(101), "orders_202604", order.OrderNo, order.CreatedAt)
		.WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_detail(order_id, item_type, sku_id, name, flavor, quantity, unit_amount, line_amount, create_time) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)"))
		.WithArgs(int64(101), "dish", int64(1), "A", "", int64(2), int64(600), int64(1200), order.CreatedAt)
		.WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.SaveOrder(context.Background(), order, items); err != nil {
		t.Fatalf("SaveOrder failed: %v", err)
	}
	if order.OrderID != 101 {
		t.Fatalf("expected order id 101, got %d", order.OrderID)
	}
}

func TestMySQLOrderRepo_UpdateWithVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLOrderRepo(db, nil)
	order := &domain.Order{OrderID: 101, Status: domain.OrderStatusCanceled, TotalAmount: 1200, Remark: "r", Version: 2, UpdatedAt: time.Now()}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT table_name FROM order_table_index WHERE order_id=? LIMIT 1")).WithArgs(int64(101)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}).AddRow("orders_202604"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE orders_202604 SET status=?, amount=?, remark=?, version=?, update_time=? WHERE id=? AND version=?")).
		WithArgs(toDBStatus(order.Status), order.TotalAmount, order.Remark, order.Version, order.UpdatedAt, order.OrderID, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.UpdateWithVersion(context.Background(), order, 1)
	if err != nil {
		t.Fatalf("UpdateWithVersion failed: %v", err)
	}
	if !ok {
		t.Fatal("expected update success")
	}
}
