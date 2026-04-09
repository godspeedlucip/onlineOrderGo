package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
	paymenttx "go-baseline-skeleton/internal/payment_callback/infra/tx"
)

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// EventPublisher persists order-paid events into outbox in the same DB transaction.
type EventPublisher struct {
	db *sql.DB
}

func NewEventPublisher(db *sql.DB) *EventPublisher {
	return &EventPublisher{db: db}
}

func (p *EventPublisher) PublishOrderPaid(ctx context.Context, evt domain.OrderPaidEvent) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("outbox db is not initialized")
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	eventID := fmt.Sprintf("%s|%s", evt.OrderNo, evt.TransactionNo)
	now := time.Now()
	_, err = p.execer(ctx).ExecContext(ctx,
		"INSERT INTO payment_outbox(event_id, event_type, payload, status, retry_count, next_retry_at, created_at, updated_at) "+
			"VALUES (?, 'ORDER_PAID', ?, 'PENDING', 0, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)",
		eventID, payload, now, now, now,
	)
	return err
}

func (p *EventPublisher) execer(ctx context.Context) sqlExecer {
	if tx, ok := paymenttx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return p.db
}
