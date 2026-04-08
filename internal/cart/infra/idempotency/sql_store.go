package idempotency

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Acquire(ctx context.Context, scene, key string, ttl time.Duration) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, fmt.Errorf("sql db is not initialized")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now()
	token := fmt.Sprintf("%d", now.UnixNano())
	expireAt := now.Add(ttl)

	res, err := s.db.ExecContext(ctx,
		"INSERT INTO cart_idempotency(scene, idem_key, token, status, result_blob, reason, updated_at, expire_at) VALUES (?, ?, ?, 'PROCESSING', NULL, '', ?, ?) "+
			"ON DUPLICATE KEY UPDATE scene = scene",
		scene, key, token, now, expireAt,
	)
	if err != nil {
		return "", false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 1 {
		return token, true, nil
	}
	// Existing row found. Try to take over only if the previous record is already expired.
	res, err = s.db.ExecContext(ctx,
		"UPDATE cart_idempotency SET token=?, status='PROCESSING', result_blob=NULL, reason='', updated_at=?, expire_at=? "+
			"WHERE scene=? AND idem_key=? AND expire_at<=?",
		token, now, expireAt, scene, key, now,
	)
	if err != nil {
		return "", false, err
	}
	affected, _ = res.RowsAffected()
	if affected > 0 {
		return token, true, nil
	}
	return "", false, nil
}

func (s *SQLStore) MarkDone(ctx context.Context, scene, key, token string, result []byte) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sql db is not initialized")
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE cart_idempotency SET status='DONE', result_blob=?, reason='', updated_at=? WHERE scene=? AND idem_key=? AND token=? AND status='PROCESSING'",
		result, time.Now(), scene, key, token,
	)
	return err
}

func (s *SQLStore) MarkFailed(ctx context.Context, scene, key, token, reason string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sql db is not initialized")
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM cart_idempotency WHERE scene=? AND idem_key=? AND token=? AND status='PROCESSING'",
		scene, key, token,
	)
	if err != nil {
		return err
	}
	_ = reason
	return nil
}

func (s *SQLStore) GetDoneResult(ctx context.Context, scene, key string) ([]byte, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("sql db is not initialized")
	}
	var status string
	var payload []byte
	var expireAt time.Time
	err := s.db.QueryRowContext(ctx,
		"SELECT status, result_blob, expire_at FROM cart_idempotency WHERE scene=? AND idem_key=? LIMIT 1",
		scene, key,
	).Scan(&status, &payload, &expireAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(expireAt) {
		_, _ = s.db.ExecContext(ctx, "DELETE FROM cart_idempotency WHERE scene=? AND idem_key=?", scene, key)
		return nil, false, nil
	}
	if strings.ToUpper(status) != "DONE" {
		return nil, false, nil
	}
	return payload, true, nil
}
