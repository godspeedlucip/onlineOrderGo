package mq

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

func TestEventPublisher_PublishOrderPaid_EnqueueOutbox(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	pub := NewEventPublisher(db)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO payment_outbox(event_id, event_type, payload, status, retry_count, next_retry_at, created_at, updated_at) VALUES (?, 'ORDER_PAID', ?, 'PENDING', 0, ?, ?, ?) ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)")).
		WithArgs("ORDER_1|TXN_1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = pub.PublishOrderPaid(context.Background(), domain.OrderPaidEvent{
		OrderID:       1,
		OrderNo:       "ORDER_1",
		TransactionNo: "TXN_1",
		PaidAmount:    1000,
		PaidAt:        time.Now(),
	})
	if err != nil {
		t.Fatalf("PublishOrderPaid failed: %v", err)
	}
}
