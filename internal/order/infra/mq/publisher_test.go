package mq

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/order/domain"
)

func TestPublisher_PublishOrderEvent_OutboxInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	pub := NewPublisher(db)
	evt := domain.OrderEvent{Type: domain.OrderEventCreated, OrderID: 101, OrderNo: "O1", To: domain.OrderStatusPendingPay, OccurredAt: time.Now()}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_outbox(event_id, event_type, order_id, order_no, payload, status, retry_count, next_retry_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'PENDING', 0, ?, ?, ?) ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)"))
		.WithArgs(sqlmock.AnyArg(), string(evt.Type), evt.OrderID, evt.OrderNo, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())
		.WillReturnResult(sqlmock.NewResult(1, 1))

	if err := pub.PublishOrderEvent(context.Background(), evt); err != nil {
		t.Fatalf("PublishOrderEvent failed: %v", err)
	}
}
