package tx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"go-baseline-skeleton/internal/baseline/domain"
)

type txDriverState struct {
	mu           sync.Mutex
	beginCount   int
	commitCount  int
	rollbackCount int
	failBegin    bool
	failCommit   bool
	failRollback bool
}

func (s *txDriverState) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.beginCount = 0
	s.commitCount = 0
	s.rollbackCount = 0
	s.failBegin = false
	s.failCommit = false
	s.failRollback = false
}

var (
	registerTxStubDriver sync.Once
	txStubState          = &txDriverState{}
)

type txStubDriver struct{}

type txStubConn struct{}

type txStubTx struct{}

func (d *txStubDriver) Open(name string) (driver.Conn, error) {
	return &txStubConn{}, nil
}

func (c *txStubConn) Prepare(query string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *txStubConn) Close() error { return nil }
func (c *txStubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *txStubConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	txStubState.mu.Lock()
	defer txStubState.mu.Unlock()
	if txStubState.failBegin {
		return nil, errors.New("begin failed")
	}
	txStubState.beginCount++
	return &txStubTx{}, nil
}

func (t *txStubTx) Commit() error {
	txStubState.mu.Lock()
	defer txStubState.mu.Unlock()
	if txStubState.failCommit {
		return errors.New("commit failed")
	}
	txStubState.commitCount++
	return nil
}

func (t *txStubTx) Rollback() error {
	txStubState.mu.Lock()
	defer txStubState.mu.Unlock()
	if txStubState.failRollback {
		return errors.New("rollback failed")
	}
	txStubState.rollbackCount++
	return nil
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	registerTxStubDriver.Do(func() {
		sql.Register("baseline_tx_stub", &txStubDriver{})
	})
	db, err := sql.Open("baseline_tx_stub", "")
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	return db
}

func TestSQLManager_RunInTx_CommitOnSuccess(t *testing.T) {
	txStubState.reset()
	db := newTestDB(t)
	defer db.Close()

	manager := NewSQLManager(db, nil)
	err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatal("expected tx in context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunInTx unexpected error: %v", err)
	}

	txStubState.mu.Lock()
	defer txStubState.mu.Unlock()
	if txStubState.beginCount != 1 || txStubState.commitCount != 1 || txStubState.rollbackCount != 0 {
		t.Fatalf("unexpected tx counters: begin=%d commit=%d rollback=%d", txStubState.beginCount, txStubState.commitCount, txStubState.rollbackCount)
	}
}

func TestSQLManager_RunInTx_RollbackOnBizError(t *testing.T) {
	txStubState.reset()
	db := newTestDB(t)
	defer db.Close()

	manager := NewSQLManager(db, nil)
	expected := errors.New("biz failed")
	err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected biz error, got: %v", err)
	}

	txStubState.mu.Lock()
	defer txStubState.mu.Unlock()
	if txStubState.beginCount != 1 || txStubState.commitCount != 0 || txStubState.rollbackCount != 1 {
		t.Fatalf("unexpected tx counters: begin=%d commit=%d rollback=%d", txStubState.beginCount, txStubState.commitCount, txStubState.rollbackCount)
	}
}

func TestSQLManager_RunInTx_RollbackFailureReturnsBizError(t *testing.T) {
	txStubState.reset()
	txStubState.failRollback = true
	db := newTestDB(t)
	defer db.Close()

	manager := NewSQLManager(db, nil)
	err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
		return errors.New("biz failed")
	})

	var bizErr *domain.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected BizError, got: %T %v", err, err)
	}
	if bizErr.Code != domain.CodeServiceUnavailable {
		t.Fatalf("unexpected biz code: %s", bizErr.Code)
	}
}
