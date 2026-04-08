package tx

import (
	"context"
	"database/sql"

	"go-baseline-skeleton/internal/baseline/domain"
)

type txContextKey string

const contextKeyTx txContextKey = "baseline_sql_tx"

type SQLManager struct {
	db      *sql.DB
	txOpts  *sql.TxOptions
}

func NewSQLManager(db *sql.DB, txOpts *sql.TxOptions) *SQLManager {
	return &SQLManager{db: db, txOpts: txOpts}
}

func (m *SQLManager) BeginTx(ctx context.Context) (context.Context, *sql.Tx, error) {
	if m == nil || m.db == nil {
		return ctx, nil, domain.NewBizError(domain.CodeServiceUnavailable, "tx db is not initialized", nil)
	}
	tx, err := m.db.BeginTx(ctx, m.txOpts)
	if err != nil {
		return ctx, nil, domain.NewBizError(domain.CodeServiceUnavailable, "begin transaction failed", err)
	}
	return context.WithValue(ctx, contextKeyTx, tx), tx, nil
}

func (m *SQLManager) Commit(tx *sql.Tx) error {
	if tx == nil {
		return domain.NewBizError(domain.CodeInternal, "commit transaction failed: tx is nil", nil)
	}
	if err := tx.Commit(); err != nil {
		return domain.NewBizError(domain.CodeServiceUnavailable, "commit transaction failed", err)
	}
	return nil
}

func (m *SQLManager) Rollback(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		return domain.NewBizError(domain.CodeServiceUnavailable, "rollback transaction failed", err)
	}
	return nil
}

func (m *SQLManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if fn == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "transaction callback is required", nil)
	}
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	txCtx, tx, beginErr := m.BeginTx(ctx)
	if beginErr != nil {
		return beginErr
	}

	defer func() {
		if rec := recover(); rec != nil {
			_ = m.Rollback(tx)
			panic(rec)
		}
		if err != nil {
			rbErr := m.Rollback(tx)
			if rbErr != nil {
				err = domain.NewBizError(domain.CodeServiceUnavailable, "transaction failed and rollback failed", rbErr)
			}
			return
		}

		if commitErr := m.Commit(tx); commitErr != nil {
			err = commitErr
		}
	}()

	err = fn(txCtx)
	return err
}

func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	v := ctx.Value(contextKeyTx)
	tx, ok := v.(*sql.Tx)
	return tx, ok
}
