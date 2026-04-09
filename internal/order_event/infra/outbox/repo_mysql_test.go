package outbox

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/order_event/domain"
)

func TestMySQLRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepo(db)
	evt := domain.OrderEvent{EventID: "E1", EventType: domain.EventOrderCreated, OrderID: 1, OrderNo: "O1", OccurredAt: time.Now(), Version: 1}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO order_event_outbox(event_id, event_type, order_id, order_no, payload, status, retry_count, next_retry_at, last_error, published_at, created_at, updated_at) VALUES(?, ?, ?, ?, ?, 'PENDING', 0, ?, '', NULL, ?, ?) ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)"))
		.WithArgs(evt.EventID, string(evt.EventType), evt.OrderID, evt.OrderNo, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())
		.WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Save(context.Background(), evt); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestMySQLRepo_FetchPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepo(db)
	payload := `{"eventId":"E2","eventType":"ORDER_CANCELED","orderId":2,"orderNo":"O2","occurredAt":"2026-04-09T01:02:03Z","version":1}`
	mock.ExpectQuery(regexp.QuoteMeta("SELECT payload FROM order_event_outbox WHERE status IN ('PENDING','FAILED') AND next_retry_at <= ? ORDER BY id ASC LIMIT ?"))
		.WithArgs(sqlmock.AnyArg(), 10)
		.WillReturnRows(sqlmock.NewRows([]string{"payload"}).AddRow(payload))

	out, err := repo.FetchPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchPending failed: %v", err)
	}
	if len(out) != 1 || out[0].EventID != "E2" {
		t.Fatalf("unexpected outbox events: %+v", out)
	}
}

func TestMySQLRepo_MarkPublishedAndFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLRepo(db)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE order_event_outbox SET status='PUBLISHED', published_at=?, last_error='', updated_at=? WHERE event_id=?"))
		.WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "E3")
		.WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkPublished(context.Background(), "E3", time.Now()); err != nil {
		t.Fatalf("MarkPublished failed: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE order_event_outbox SET status='FAILED', retry_count=retry_count+1, next_retry_at=?, last_error=?, updated_at=? WHERE event_id=?"))
		.WithArgs(sqlmock.AnyArg(), "publish fail", sqlmock.AnyArg(), "E3")
		.WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkFailed(context.Background(), "E3", "publish fail", time.Now().Add(time.Second)); err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}
}
