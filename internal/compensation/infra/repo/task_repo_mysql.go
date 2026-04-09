package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type MonthTableRouter interface {
	CandidateTables(ctx context.Context, anchor time.Time) ([]string, error)
}

type MySQLTaskRepo struct {
	db          *sql.DB
	router      MonthTableRouter
	backoffBase time.Duration
	backoffMax  time.Duration
	claimTTL    time.Duration
}

func NewMySQLTaskRepo(db *sql.DB, router MonthTableRouter) *MySQLTaskRepo {
	return &MySQLTaskRepo{
		db:          db,
		router:      router,
		backoffBase: 5 * time.Second,
		backoffMax:  10 * time.Minute,
		claimTTL:    30 * time.Second,
	}
}

func (r *MySQLTaskRepo) SaveRun(ctx context.Context, rec domain.TaskRunRecord) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO compensation_task_run(task_id, job_type, status, reason, retry_count, started_at, finished_at, duration_ms, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
		rec.TaskID, string(rec.JobType), string(rec.Status), rec.Reason, rec.RetryCount, rec.StartedAt, rec.FinishedAt, rec.DurationMs, time.Now(),
	)
	return err
}

func (r *MySQLTaskRepo) MarkDone(ctx context.Context, taskID string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		"UPDATE compensation_task SET status='DONE', last_error='', dead_reason='', updated_at=? WHERE task_id=? AND status IN ('PENDING','FAILED','RUNNING')",
		time.Now(), taskID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewBizError(domain.CodeConflict, "task mark done skipped by condition", nil)
	}
	return nil
}

func (r *MySQLTaskRepo) MarkFailed(ctx context.Context, taskID, reason string) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	var (
		retryCount int
		maxRetry   int
	)
	err := r.db.QueryRowContext(ctx, "SELECT retry_count, max_retry FROM compensation_task WHERE task_id=? LIMIT 1", taskID).Scan(&retryCount, &maxRetry)
	if err == sql.ErrNoRows {
		return domain.NewBizError(domain.CodeInvalidArgument, "task not found", nil)
	}
	if err != nil {
		return err
	}
	nextCount := retryCount + 1
	now := time.Now()
	if nextCount > maxRetry {
		res, upErr := r.db.ExecContext(ctx,
			"UPDATE compensation_task SET status='DEAD', dead_reason=?, last_error=?, updated_at=?, dead_at=? WHERE task_id=? AND retry_count=? AND status IN ('PENDING','FAILED','RUNNING')",
			reason, reason, now, now, taskID, retryCount,
		)
		if upErr != nil {
			return upErr
		}
		affected, rowsErr := res.RowsAffected()
		if rowsErr != nil {
			return rowsErr
		}
		if affected == 0 {
			return domain.NewBizError(domain.CodeConflict, "task dead-letter skipped by condition", nil)
		}
		_, _ = r.db.ExecContext(ctx,
			"INSERT INTO compensation_task_dlq(task_id, reason, archived_at) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE reason=VALUES(reason), archived_at=VALUES(archived_at)",
			taskID, reason, now,
		)
		return nil
	}
	nextAt := now.Add(r.backoffAt(nextCount))
	res, err := r.db.ExecContext(ctx,
		"UPDATE compensation_task SET status='FAILED', retry_count=?, next_execute_at=?, last_error=?, updated_at=? WHERE task_id=? AND retry_count=? AND status IN ('PENDING','FAILED','RUNNING')",
		nextCount, nextAt, reason, now, taskID, retryCount,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.NewBizError(domain.CodeConflict, "task retry update skipped by condition", nil)
	}
	return nil
}

func (r *MySQLTaskRepo) Scan(ctx context.Context, job domain.JobType, limit int) ([]domain.TaskItem, error) {
	if strings.TrimSpace(string(job)) == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty job type", nil)
	}
	return r.scanAndClaim(ctx, "job_type=?", []any{string(job)}, limit)
}

func (r *MySQLTaskRepo) ScanFailed(ctx context.Context, limit int) ([]domain.TaskItem, error) {
	return r.scanAndClaim(ctx, "status='FAILED'", nil, limit)
}

func (r *MySQLTaskRepo) scanAndClaim(ctx context.Context, extraCond string, extraArgs []any, limit int) ([]domain.TaskItem, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	now := time.Now()
	conds := []string{"status IN ('PENDING','FAILED')", "next_execute_at<=?"}
	args := []any{now}
	if strings.TrimSpace(extraCond) != "" {
		conds = append(conds, extraCond)
		args = append(args, extraArgs...)
	}
	candidates, routeErr := r.resolveCandidateTables(ctx, now)
	if routeErr != nil {
		return nil, routeErr
	}
	if len(candidates) == 0 {
		conds = append(conds, "(shard_table IS NULL OR shard_table='')")
	} else {
		holders := make([]string, 0, len(candidates))
		for _, t := range candidates {
			holders = append(holders, "?")
			args = append(args, t)
		}
		conds = append(conds, "(shard_table IS NULL OR shard_table='' OR shard_table IN ("+strings.Join(holders, ",")+"))")
	}
	args = append(args, limit)
	query := "SELECT task_id, job_type, biz_key, payload, scheduled_at, retry_count, max_retry, next_execute_at, dead_reason, shard_table FROM compensation_task WHERE " +
		strings.Join(conds, " AND ") +
		" ORDER BY next_execute_at ASC LIMIT ?"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	picked := make([]domain.TaskItem, 0, limit)
	for rows.Next() {
		var (
			item domain.TaskItem
			job  string
		)
		if scanErr := rows.Scan(&item.TaskID, &job, &item.BizKey, &item.Payload, &item.ScheduledAt, &item.RetryCount, &item.MaxRetry, &item.NextExecuteAt, &item.DeadReason, &item.ShardTable); scanErr != nil {
			return nil, scanErr
		}
		item.JobType = domain.JobType(strings.TrimSpace(job))
		claimToken := fmt.Sprintf("%d", time.Now().UnixNano())
		res, claimErr := r.db.ExecContext(ctx,
			"UPDATE compensation_task SET status='RUNNING', lock_token=?, lock_expire_at=?, updated_at=? WHERE task_id=? AND status IN ('PENDING','FAILED')",
			claimToken, time.Now().Add(r.claimTTL), time.Now(), item.TaskID,
		)
		if claimErr != nil {
			return nil, claimErr
		}
		affected, rowsErr := res.RowsAffected()
		if rowsErr != nil {
			return nil, rowsErr
		}
		if affected == 0 {
			continue
		}
		item.ClaimToken = claimToken
		picked = append(picked, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return picked, nil
}

func (r *MySQLTaskRepo) resolveCandidateTables(ctx context.Context, anchor time.Time) ([]string, error) {
	if r.router == nil {
		return nil, nil
	}
	return r.router.CandidateTables(ctx, anchor)
}

func (r *MySQLTaskRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "compensation db is not initialized", nil)
	}
	return nil
}

func (r *MySQLTaskRepo) backoffAt(nextRetryCount int) time.Duration {
	if nextRetryCount <= 1 {
		return r.backoffBase
	}
	wait := r.backoffBase
	for i := 2; i <= nextRetryCount; i++ {
		wait = wait * 2
		if wait >= r.backoffMax {
			return r.backoffMax
		}
	}
	if wait > r.backoffMax {
		return r.backoffMax
	}
	return wait
}
