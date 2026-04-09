package repo

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go-baseline-skeleton/internal/compensation/domain"
)

type fakeMonthRouter struct {
	tables []string
}

func (r *fakeMonthRouter) CandidateTables(ctx context.Context, anchor time.Time) ([]string, error) {
	_ = ctx
	_ = anchor
	return r.tables, nil
}

func TestMySQLTaskRepo_MarkDone_Conditional(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLTaskRepo(db, nil)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE compensation_task SET status='DONE', last_error='', dead_reason='', updated_at=? WHERE task_id=? AND status IN ('PENDING','FAILED','RUNNING')")).
		WithArgs(sqlmock.AnyArg(), "t1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkDone(context.Background(), "t1"); err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestMySQLTaskRepo_MarkFailed_RetryBackoff(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLTaskRepo(db, nil)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT retry_count, max_retry FROM compensation_task WHERE task_id=? LIMIT 1")).
		WithArgs("t2").
		WillReturnRows(sqlmock.NewRows([]string{"retry_count", "max_retry"}).AddRow(0, 3))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE compensation_task SET status='FAILED', retry_count=?, next_execute_at=?, last_error=?, updated_at=? WHERE task_id=? AND retry_count=? AND status IN ('PENDING','FAILED','RUNNING')")).
		WithArgs(1, sqlmock.AnyArg(), "boom", sqlmock.AnyArg(), "t2", 0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkFailed(context.Background(), "t2", "boom"); err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestMySQLTaskRepo_MarkFailed_ArchiveDeadLetter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLTaskRepo(db, nil)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT retry_count, max_retry FROM compensation_task WHERE task_id=? LIMIT 1")).
		WithArgs("t3").
		WillReturnRows(sqlmock.NewRows([]string{"retry_count", "max_retry"}).AddRow(3, 2))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE compensation_task SET status='DEAD', dead_reason=?, last_error=?, updated_at=?, dead_at=? WHERE task_id=? AND retry_count=? AND status IN ('PENDING','FAILED','RUNNING')")).
		WithArgs("fatal", "fatal", sqlmock.AnyArg(), sqlmock.AnyArg(), "t3", 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO compensation_task_dlq(task_id, reason, archived_at) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE reason=VALUES(reason), archived_at=VALUES(archived_at)")).
		WithArgs("t3", "fatal", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkFailed(context.Background(), "t3", "fatal"); err != nil {
		t.Fatalf("MarkFailed dead letter failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestMySQLTaskRepo_Scan_ClaimWithShardCandidate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLTaskRepo(db, &fakeMonthRouter{tables: []string{"orders_202604", "orders_202603"}})
	mock.ExpectQuery("SELECT task_id, job_type, biz_key, payload, scheduled_at, retry_count, max_retry, next_execute_at, dead_reason, shard_table FROM compensation_task WHERE .*shard_table IN").
		WithArgs(sqlmock.AnyArg(), "ORDER_TIMEOUT_CANCEL", "orders_202604", "orders_202603", 10).
		WillReturnRows(sqlmock.NewRows([]string{"task_id", "job_type", "biz_key", "payload", "scheduled_at", "retry_count", "max_retry", "next_execute_at", "dead_reason", "shard_table"}).
			AddRow("task_1", "ORDER_TIMEOUT_CANCEL", "order:1001", []byte(`{"orderId":1001}`), time.Now().Add(-time.Minute), 0, 3, time.Now().Add(-time.Minute), "", "orders_202604"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE compensation_task SET status='RUNNING', lock_token=?, lock_expire_at=?, updated_at=? WHERE task_id=? AND status IN ('PENDING','FAILED')")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "task_1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	items, err := repo.Scan(context.Background(), domain.JobOrderTimeoutCancel, 10)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(items) != 1 || items[0].TaskID != "task_1" || items[0].ClaimToken == "" {
		t.Fatalf("unexpected scan items: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
