package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go-baseline-skeleton/internal/order/domain"
	ordertx "go-baseline-skeleton/internal/order/infra/tx"
)

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Publisher struct {
	db *sql.DB
}

func NewPublisher(db *sql.DB) *Publisher { return &Publisher{db: db} }

func (p *Publisher) PublishOrderEvent(ctx context.Context, evt domain.OrderEvent) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("outbox db is not initialized")
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	eventID := fmt.Sprintf("%s|%d|%d", evt.Type, evt.OrderID, evt.OccurredAt.UnixNano())
	now := time.Now()
	_, err = p.execer(ctx).ExecContext(ctx,
		"INSERT INTO order_outbox(event_id, event_type, order_id, order_no, payload, status, retry_count, next_retry_at, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, 'PENDING', 0, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)",
		eventID, string(evt.Type), evt.OrderID, evt.OrderNo, payload, now, now, now,
	)
	return err
}

func (p *Publisher) execer(ctx context.Context) sqlExecer {
	if tx, ok := ordertx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return p.db
}
