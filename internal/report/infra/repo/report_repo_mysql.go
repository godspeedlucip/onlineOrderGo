package repo

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"go-baseline-skeleton/internal/report/domain"
)

var tableNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type MySQLReportRepo struct {
	db *sql.DB
}

func NewMySQLReportRepo(db *sql.DB) *MySQLReportRepo {
	return &MySQLReportRepo{db: db}
}

func (r *MySQLReportRepo) QueryOverviewFromTable(ctx context.Context, table string, q domain.OverviewQuery) (*domain.OverviewPartial, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if !isSafeTableName(table) {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid table name", nil)
	}

	query := fmt.Sprintf(
		"SELECT "+
			"COUNT(id) AS order_count, "+
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS valid_order_count, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN amount ELSE 0 END), 0) AS turnover, "+
			"COALESCE(SUM(CASE WHEN pay_status = ? THEN amount ELSE 0 END), 0) AS refund_amount, "+
			"COUNT(DISTINCT user_id) AS user_count "+
			"FROM %s WHERE order_time >= ? AND order_time <= ?",
		table,
	)
	args := []any{
		domain.OrderStatusCompleted,
		domain.OrderStatusCompleted,
		domain.PayStatusRefund,
		q.From,
		q.To,
	}
	if q.StoreID > 0 {
		query += " AND store_id = ?"
		args = append(args, q.StoreID)
	}

	out := &domain.OverviewPartial{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&out.OrderCount,
		&out.ValidOrderCount,
		&out.Turnover,
		&out.RefundAmount,
		&out.UserCount,
	)
	if err != nil {
		if isTableNotExistsErr(err) {
			return &domain.OverviewPartial{}, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *MySQLReportRepo) QueryTrendFromTable(ctx context.Context, table string, q domain.TrendQuery) ([]domain.TrendPoint, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if !isSafeTableName(table) {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid table name", nil)
	}

	groupExpr, timeAlias := toTimeGroupExpr(q.Granularity)
	query := fmt.Sprintf(
		"SELECT %s AS %s, COALESCE(SUM(amount), 0) AS v "+
			"FROM %s WHERE order_time >= ? AND order_time <= ? AND status = ?",
		groupExpr, timeAlias, table,
	)
	args := []any{q.From, q.To, domain.OrderStatusCompleted}
	if q.StoreID > 0 {
		query += " AND store_id = ?"
		args = append(args, q.StoreID)
	}
	query += fmt.Sprintf(" GROUP BY %s ORDER BY %s ASC", groupExpr, groupExpr)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		if isTableNotExistsErr(err) {
			return []domain.TrendPoint{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TrendPoint, 0)
	for rows.Next() {
		var p domain.TrendPoint
		if scanErr := rows.Scan(&p.TimeKey, &p.Value); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		if isTableNotExistsErr(err) {
			return []domain.TrendPoint{}, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *MySQLReportRepo) QueryOrdersFromTable(ctx context.Context, table string, q domain.OrderListQuery) ([]domain.OrderRow, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if !isSafeTableName(table) {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid table name", nil)
	}

	query := fmt.Sprintf(
		"SELECT id, number, status, amount, order_time "+
			"FROM %s WHERE order_time >= ? AND order_time <= ?",
		table,
	)
	args := []any{q.From, q.To}
	if q.StoreID > 0 {
		query += " AND store_id = ?"
		args = append(args, q.StoreID)
	}
	query += " ORDER BY order_time DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		if isTableNotExistsErr(err) {
			return []domain.OrderRow{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.OrderRow, 0)
	for rows.Next() {
		var row domain.OrderRow
		if scanErr := rows.Scan(&row.OrderID, &row.OrderNumber, &row.Status, &row.Amount, &row.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		if isTableNotExistsErr(err) {
			return []domain.OrderRow{}, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *MySQLReportRepo) ensureDB() error {
	if r == nil || r.db == nil {
		return domain.NewBizError(domain.CodeInternal, "report db is not initialized", nil)
	}
	return nil
}

func isSafeTableName(table string) bool {
	return tableNamePattern.MatchString(strings.TrimSpace(table))
}

func isTableNotExistsErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "error 1146")
}

func toTimeGroupExpr(granularity string) (expr string, alias string) {
	switch strings.ToLower(strings.TrimSpace(granularity)) {
	case "month":
		return "DATE_FORMAT(order_time, '%Y-%m')", "time_key"
	case "hour":
		return "DATE_FORMAT(order_time, '%Y-%m-%d %H')", "time_key"
	default:
		return "DATE_FORMAT(order_time, '%Y-%m-%d')", "time_key"
	}
}
