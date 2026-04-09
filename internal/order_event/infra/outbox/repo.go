package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
	eventtx "go-baseline-skeleton/internal/order_event/infra/tx"
)

type item struct {
	evt         domain.OrderEvent
	publishedAt *time.Time
	nextRetryAt time.Time
	retryCount  int
	reason      string
}

type InMemoryRepo struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{items: make(map[string]item)}
}

func (r *InMemoryRepo) Save(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	r.mu.Lock()
	r.items[evt.EventID] = item{evt: evt, nextRetryAt: time.Now()}
	r.mu.Unlock()
	return nil
}

func (r *InMemoryRepo) FetchPending(ctx context.Context, limit int) ([]domain.OrderEvent, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	now := time.Now()
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.OrderEvent, 0, limit)
	for _, it := range r.items {
		if it.publishedAt != nil {
			continue
		}
		if !it.nextRetryAt.IsZero() && it.nextRetryAt.After(now) {
			continue
		}
		out = append(out, it.evt)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *InMemoryRepo) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	_ = ctx
	r.mu.Lock()
	old, ok := r.items[eventID]
	if ok {
		old.publishedAt = &publishedAt
		r.items[eventID] = old
	}
	r.mu.Unlock()
	return nil
}

func (r *InMemoryRepo) MarkFailed(ctx context.Context, eventID, reason string, nextRetryAt time.Time) error {
	_ = ctx
	r.mu.Lock()
	old, ok := r.items[eventID]
	if ok {
		old.retryCount++
		old.reason = reason
		old.nextRetryAt = nextRetryAt
		r.items[eventID] = old
	}
	r.mu.Unlock()
	return nil
}

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type MySQLRepo struct {
	db *sql.DB
}

func NewMySQLRepo(db *sql.DB) *MySQLRepo {
	return &MySQLRepo{db: db}
}

func (r *MySQLRepo) Save(ctx context.Context, evt domain.OrderEvent) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = r.execer(ctx).ExecContext(ctx,
		"INSERT INTO order_event_outbox(event_id, event_type, order_id, order_no, payload, status, retry_count, next_retry_at, last_error, published_at, created_at, updated_at) "+
			"VALUES(?, ?, ?, ?, ?, 'PENDING', 0, ?, '', NULL, ?, ?) "+
			"ON DUPLICATE KEY UPDATE payload=VALUES(payload), updated_at=VALUES(updated_at)",
		evt.EventID, string(evt.EventType), evt.OrderID, evt.OrderNo, payload, now, now, now,
	)
	return err
}

func (r *MySQLRepo) FetchPending(ctx context.Context, limit int) ([]domain.OrderEvent, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.execer(ctx).QueryContext(ctx,
		"SELECT payload FROM order_event_outbox WHERE status IN ('PENDING','FAILED') AND next_retry_at <= ? ORDER BY id ASC LIMIT ?",
		time.Now(), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OrderEvent, 0)
	for rows.Next() {
		var payload []byte
		if scanErr := rows.Scan(&payload); scanErr != nil {
			return nil, scanErr
		}
		var evt domain.OrderEvent
		if unmarshalErr := json.Unmarshal(payload, &evt); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		out = append(out, evt)
	}
	return out, rows.Err()
}

func (r *MySQLRepo) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE order_event_outbox SET status='PUBLISHED', published_at=?, last_error='', updated_at=? WHERE event_id=?",
		publishedAt, time.Now(), eventID,
	)
	return err
}

func (r *MySQLRepo) MarkFailed(ctx context.Context, eventID, reason string, nextRetryAt time.Time) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx,
		"UPDATE order_event_outbox SET status='FAILED', retry_count=retry_count+1, next_retry_at=?, last_error=?, updated_at=? WHERE event_id=?",
		nextRetryAt, reason, time.Now(), eventID,
	)
	return err
}

func (r *MySQLRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "order event outbox db is not initialized", nil)
	}
	return nil
}

func (r *MySQLRepo) execer(ctx context.Context) sqlExecQuerier {
	if tx, ok := eventtx.TxFromContext(ctx); ok && tx != nil {
		return tx
	}
	return r.db
}
