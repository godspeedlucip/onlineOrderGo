package tx

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLManager_RunInTx_CommitOnSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	m := NewSQLManager(db, nil)
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = m.RunInTx(context.Background(), func(ctx context.Context) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatal("tx missing in context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunInTx failed: %v", err)
	}
}

func TestSQLManager_RunInTx_RollbackOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	m := NewSQLManager(db, nil)
	mock.ExpectBegin()
	mock.ExpectRollback()
	want := errors.New("save outbox failed")
	err = m.RunInTx(context.Background(), func(ctx context.Context) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("expected %v got %v", want, err)
	}
}
