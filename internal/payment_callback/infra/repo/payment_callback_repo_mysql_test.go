package repo

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMySQLCallbackRepo_UpdateOrderPaidIfPending_Conditional(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLCallbackRepo(db)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE orders SET status=?, pay_status=?, checkout_time=?, update_time=? WHERE id=? AND status=? AND pay_status=?")).
		WithArgs(orderStatusToBeConfirmed, payStatusPaid, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1001), orderStatusPendingPayment, payStatusUnPaid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.UpdateOrderPaidIfPending(context.Background(), 1001, time.Now(), "txn-1", 6800)
	if err != nil {
		t.Fatalf("UpdateOrderPaidIfPending failed: %v", err)
	}
	if !ok {
		t.Fatal("expected updated=true")
	}
}

func TestMySQLCallbackRepo_GetOrderByNo(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLCallbackRepo(db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, number, status, amount, merchant_id FROM orders WHERE number=? LIMIT 1")).
		WithArgs("ORDER_1001").
		WillReturnRows(sqlmock.NewRows([]string{"id", "number", "status", "amount", "merchant_id"}).AddRow(1001, "ORDER_1001", 1, 6800, "M001"))

	order, err := repo.GetOrderByNo(context.Background(), "ORDER_1001")
	if err != nil {
		t.Fatalf("GetOrderByNo failed: %v", err)
	}
	if order == nil || order.OrderID != 1001 || order.TotalAmount != 6800 {
		t.Fatalf("unexpected order: %+v", order)
	}
}
