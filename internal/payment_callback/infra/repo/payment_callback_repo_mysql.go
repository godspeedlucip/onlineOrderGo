package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
	paymenttx "go-baseline-skeleton/internal/payment_callback/infra/tx"
)

const (
	orderStatusPendingPayment = 1
	orderStatusToBeConfirmed  = 2
	payStatusUnPaid           = 0
	payStatusPaid             = 1
)

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type MySQLCallbackRepo struct {
	db *sql.DB
}

func NewMySQLCallbackRepo(db *sql.DB) *MySQLCallbackRepo {
	return &MySQLCallbackRepo{db: db}
}

func (r *MySQLCallbackRepo) GetOrderByNo(ctx context.Context, orderNo string) (*domain.OrderSnapshot, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	var out domain.OrderSnapshot
	err := r.execer(ctx).QueryRowContext(ctx,
		"SELECT id, number, status, amount, merchant_id FROM orders WHERE number=? LIMIT 1",
		orderNo,
	).Scan(&out.OrderID, &out.OrderNo, &out.Status, &out.TotalAmount, &out.MerchantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *MySQLCallbackRepo) UpdateOrderPaidIfPending(ctx context.Context, orderID int64, payAt time.Time, txnNo string, paidAmount int64) (bool, error) {
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	_ = txnNo
	_ = paidAmount
	res, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE orders SET status=?, pay_status=?, checkout_time=?, update_time=? WHERE id=? AND status=? AND pay_status=?",
		orderStatusToBeConfirmed, payStatusPaid, payAt, time.Now(), orderID, orderStatusPendingPayment, payStatusUnPaid,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *MySQLCallbackRepo) InsertPaymentRecord(ctx context.Context, rec domain.PaymentRecord) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx,
		"INSERT INTO payment_record(order_id, order_no, transaction_no, channel, paid_amount, paid_at, raw_status, create_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE raw_status=VALUES(raw_status), paid_at=VALUES(paid_at), update_time=VALUES(update_time)",
		rec.OrderID, rec.OrderNo, rec.TransactionNo, rec.Channel, rec.PaidAmount, rec.PaidAt, rec.RawStatus, time.Now(), time.Now(),
	)
	if err != nil {
		if isDuplicateKey(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *MySQLCallbackRepo) InsertCallbackLog(ctx context.Context, log domain.CallbackLog) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	headers, body := marshalLog(log)
	_, err := r.execer(ctx).ExecContext(ctx,
		"INSERT INTO payment_callback_log(notify_id, order_no, transaction_no, channel, http_headers, body, verified, error_message, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		log.NotifyID, log.OrderNo, log.TransactionNo, log.Channel, headers, body, boolToInt(log.Verified), log.ErrorMessage, log.CreatedAt,
	)
	return err
}

func (r *MySQLCallbackRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "payment callback db is not initialized", nil)
	}
	return nil
}

func (r *MySQLCallbackRepo) execer(ctx context.Context) sqlExecQuerier {
	if tx, ok := paymenttx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func marshalLog(log domain.CallbackLog) ([]byte, []byte) {
	headers := []byte("{}")
	body := []byte{}
	if log.HTTPHeaders != nil {
		if b, err := json.Marshal(log.HTTPHeaders); err == nil {
			headers = b
		}
	}
	if len(log.Body) > 0 {
		body = append([]byte(nil), log.Body...)
	}
	return headers, body
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "unique constraint")
}
