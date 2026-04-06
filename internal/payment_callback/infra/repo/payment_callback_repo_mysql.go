package repo

import (
	"context"
	"sync"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type orderRow struct {
	OrderID      int64
	OrderNo      string
	Status       string
	TotalAmount  int64
	MerchantID   string
	PaidAt       time.Time
	TransactionNo string
}

type MySQLCallbackRepo struct {
	mu           sync.RWMutex
	ordersByNo   map[string]orderRow
	payRecords   []domain.PaymentRecord
	callbackLogs []domain.CallbackLog
}

func NewMySQLCallbackRepo() *MySQLCallbackRepo {
	return &MySQLCallbackRepo{
		ordersByNo: map[string]orderRow{
			"ORDER_1001": {OrderID: 1001, OrderNo: "ORDER_1001", Status: "PENDING", TotalAmount: 6800, MerchantID: "M001"},
			"ORDER_1002": {OrderID: 1002, OrderNo: "ORDER_1002", Status: "PAID", TotalAmount: 3200, MerchantID: "M001"},
		},
		payRecords:   make([]domain.PaymentRecord, 0),
		callbackLogs: make([]domain.CallbackLog, 0),
	}
}

func (r *MySQLCallbackRepo) GetOrderByNo(ctx context.Context, orderNo string) (*domain.OrderSnapshot, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.ordersByNo[orderNo]
	if !ok {
		return nil, nil
	}
	return &domain.OrderSnapshot{OrderID: row.OrderID, OrderNo: row.OrderNo, Status: row.Status, TotalAmount: row.TotalAmount, MerchantID: row.MerchantID}, nil
}

func (r *MySQLCallbackRepo) UpdateOrderPaidIfPending(ctx context.Context, orderID int64, payAt time.Time, txnNo string, paidAmount int64) (bool, error) {
	_ = ctx
	_ = paidAmount
	r.mu.Lock()
	defer r.mu.Unlock()
	for no, row := range r.ordersByNo {
		if row.OrderID != orderID {
			continue
		}
		if row.Status != "PENDING" {
			return false, nil
		}
		row.Status = "PAID"
		row.PaidAt = payAt
		row.TransactionNo = txnNo
		r.ordersByNo[no] = row
		return true, nil
	}
	return false, nil
}

func (r *MySQLCallbackRepo) InsertPaymentRecord(ctx context.Context, rec domain.PaymentRecord) error {
	_ = ctx
	r.mu.Lock()
	r.payRecords = append(r.payRecords, rec)
	r.mu.Unlock()
	return nil
}

func (r *MySQLCallbackRepo) InsertCallbackLog(ctx context.Context, log domain.CallbackLog) error {
	_ = ctx
	r.mu.Lock()
	r.callbackLogs = append(r.callbackLogs, log)
	r.mu.Unlock()
	return nil
}