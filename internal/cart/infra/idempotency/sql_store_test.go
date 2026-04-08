package idempotency

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLStore_AcquireAndReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart_idempotency(scene, idem_key, token, status, result_blob, reason, updated_at, expire_at) VALUES (?, ?, ?, 'PROCESSING', NULL, '', ?, ?) ON DUPLICATE KEY UPDATE scene = scene")).
		WithArgs("cart:add", "k1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	token, ok, err := store.Acquire(context.Background(), "cart:add", "k1", time.Minute)
	if err != nil || !ok || token == "" {
		t.Fatalf("Acquire failed token=%s ok=%v err=%v", token, ok, err)
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE cart_idempotency SET status='DONE', result_blob=?, reason='', updated_at=? WHERE scene=? AND idem_key=? AND token=? AND status='PROCESSING'")).
		WithArgs([]byte(`{"id":1}`), sqlmock.AnyArg(), "cart:add", "k1", token).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.MarkDone(context.Background(), "cart:add", "k1", token, []byte(`{"id":1}`)); err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, result_blob, expire_at FROM cart_idempotency WHERE scene=? AND idem_key=? LIMIT 1")).
		WithArgs("cart:add", "k1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "result_blob", "expire_at"}).AddRow("DONE", []byte(`{"id":1}`), time.Now().Add(time.Minute)))
	result, found, err := store.GetDoneResult(context.Background(), "cart:add", "k1")
	if err != nil || !found {
		t.Fatalf("GetDoneResult failed found=%v err=%v", found, err)
	}
	if string(result) != `{"id":1}` {
		t.Fatalf("unexpected result: %s", string(result))
	}
}

func TestSQLStore_AcquireTakeOverExpiredRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart_idempotency(scene, idem_key, token, status, result_blob, reason, updated_at, expire_at) VALUES (?, ?, ?, 'PROCESSING', NULL, '', ?, ?) ON DUPLICATE KEY UPDATE scene = scene")).
		WithArgs("cart:add", "k-expired", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE cart_idempotency SET token=?, status='PROCESSING', result_blob=NULL, reason='', updated_at=?, expire_at=? WHERE scene=? AND idem_key=? AND expire_at<=?")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "cart:add", "k-expired", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	token, ok, err := store.Acquire(context.Background(), "cart:add", "k-expired", time.Minute)
	if err != nil || !ok || token == "" {
		t.Fatalf("Acquire takeover failed token=%s ok=%v err=%v", token, ok, err)
	}
}
