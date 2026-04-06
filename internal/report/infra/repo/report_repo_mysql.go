package repo

import (
	"context"
	"sort"
	"strings"
	"time"

	"go-baseline-skeleton/internal/report/domain"
)

type MySQLReportRepo struct {
	store *memoryStore
}

func NewMySQLReportRepo() *MySQLReportRepo {
	return &MySQLReportRepo{store: defaultStore}
}

func (r *MySQLReportRepo) QueryOverviewFromTable(ctx context.Context, table string, q domain.OverviewQuery) (*domain.OverviewPartial, error) {
	_ = ctx
	rows := r.queryRows(table, q.From, q.To, q.StoreID)
	out := &domain.OverviewPartial{}
	seenUsers := map[int64]struct{}{}

	for _, row := range rows {
		out.OrderCount++
		if isValidOrderStatus(row.Status) {
			out.ValidOrderCount++
			out.Turnover += row.Amount
		}
		if row.Refunded {
			out.RefundAmount += row.Amount
		}
		seenUsers[row.UserID] = struct{}{}
	}
	out.UserCount = int64(len(seenUsers))
	return out, nil
}

func (r *MySQLReportRepo) QueryTrendFromTable(ctx context.Context, table string, q domain.TrendQuery) ([]domain.TrendPoint, error) {
	_ = ctx
	rows := r.queryRows(table, q.From, q.To, q.StoreID)
	bucket := map[string]int64{}

	for _, row := range rows {
		if !isValidOrderStatus(row.Status) {
			continue
		}
		key := toTimeKey(row.CreatedAt, q.Granularity)
		bucket[key] += row.Amount
	}

	keys := make([]string, 0, len(bucket))
	for k := range bucket {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]domain.TrendPoint, 0, len(keys))
	for _, k := range keys {
		out = append(out, domain.TrendPoint{TimeKey: k, Value: bucket[k]})
	}
	return out, nil
}

func (r *MySQLReportRepo) QueryOrdersFromTable(ctx context.Context, table string, q domain.OrderListQuery) ([]domain.OrderRow, error) {
	_ = ctx
	rows := r.queryRows(table, q.From, q.To, q.StoreID)
	out := make([]domain.OrderRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.OrderRow{
			OrderID:     row.OrderID,
			OrderNumber: row.OrderNumber,
			Status:      row.Status,
			Amount:      row.Amount,
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil
}

func (r *MySQLReportRepo) queryRows(table string, from, to time.Time, storeID int64) []orderRecord {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	raw := r.store.tables[table]
	out := make([]orderRecord, 0, len(raw))
	for _, row := range raw {
		if row.CreatedAt.Before(from) || row.CreatedAt.After(to) {
			continue
		}
		if storeID > 0 && row.StoreID != storeID {
			continue
		}
		out = append(out, row)
	}
	return out
}

func isValidOrderStatus(status int) bool {
	// TODO: align with Java order status set used in report SQL.
	return status == 2 || status == 3 || status == 4 || status == 5
}

func toTimeKey(t time.Time, granularity string) string {
	switch strings.ToLower(strings.TrimSpace(granularity)) {
	case "month":
		return t.Format("2006-01")
	case "hour":
		return t.Format("2006-01-02 15")
	default:
		return t.Format("2006-01-02")
	}
}