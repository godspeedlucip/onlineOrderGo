package tx

import (
	"context"
	"database/sql"

	"go-baseline-skeleton/internal/order/domain"
)

type txContextKey string

const contextKeyTx txContextKey = "order_sql_tx"

type SQLManager struct {
	db     *sql.DB
	txOpts *sql.TxOptions
}

func NewSQLManager(db *sql.DB, txOpts *sql.TxOptions) *SQLManager {
	return &SQLManager{db: db, txOpts: txOpts}
}

func (m *SQLManager) BeginTx(ctx context.Context) (context.Context, *sql.Tx, error) {
	if m == nil || m.db == nil {
		return ctx, nil, domain.NewBizError(domain.CodeInternal, "tx db is not initialized", nil)
	}
	tx, err := m.db.BeginTx(ctx, m.txOpts)
	if err != nil {
		return ctx, nil, domain.NewBizError(domain.CodeInternal, "begin tx failed", err)
	}
	return context.WithValue(ctx, contextKeyTx, tx), tx, nil
}

func (m *SQLManager) Commit(tx *sql.Tx) error {
	if tx == nil {
		return domain.NewBizError(domain.CodeInternal, "commit tx failed: tx is nil", nil)
	}
	if err := tx.Commit(); err != nil {
		return domain.NewBizError(domain.CodeInternal, "commit tx failed", err)
	}
	return nil
}

func (m *SQLManager) Rollback(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		return domain.NewBizError(domain.CodeInternal, "rollback tx failed", err)
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
			_ = m.Rollback(tx)
			return
		}
		err = m.Commit(tx)
	}()
	err = fn(txCtx)
	return err
}

func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	v := ctx.Value(contextKeyTx)
	tx, ok := v.(*sql.Tx)
	return tx, ok
}
