package repo

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/report/domain"
)

func TestMySQLReportRepo_QueryOverviewFromTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLReportRepo(db)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.Local)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(id) AS order_count, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS valid_order_count, COALESCE(SUM(CASE WHEN status = ? THEN amount ELSE 0 END), 0) AS turnover, COALESCE(SUM(CASE WHEN pay_status = ? THEN amount ELSE 0 END), 0) AS refund_amount, COUNT(DISTINCT user_id) AS user_count FROM orders_202604 WHERE order_time >= ? AND order_time <= ?")).
		WithArgs(domain.OrderStatusCompleted, domain.OrderStatusCompleted, domain.PayStatusRefund, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"order_count", "valid_order_count", "turnover", "refund_amount", "user_count"}).AddRow(12, 8, 34000, 1500, 9))

	out, err := repo.QueryOverviewFromTable(context.Background(), "orders_202604", domain.OverviewQuery{From: from, To: to})
	if err != nil {
		t.Fatalf("QueryOverviewFromTable failed: %v", err)
	}
	if out.OrderCount != 12 || out.ValidOrderCount != 8 || out.Turnover != 34000 || out.RefundAmount != 1500 || out.UserCount != 9 {
		t.Fatalf("unexpected overview: %+v", out)
	}
}

func TestMySQLReportRepo_QueryOverviewFromTable_EmptyMonthWhenTableMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLReportRepo(db)
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.Local)

	mock.ExpectQuery("SELECT COUNT\\(id\\).*FROM orders_202602").
		WithArgs(domain.OrderStatusCompleted, domain.OrderStatusCompleted, domain.PayStatusRefund, from, to).
		WillReturnError(errors.New("Error 1146 (42S02): Table 'orders_202602' doesn't exist"))

	out, err := repo.QueryOverviewFromTable(context.Background(), "orders_202602", domain.OverviewQuery{From: from, To: to})
	if err != nil {
		t.Fatalf("expected empty month as zero, got err: %v", err)
	}
	if out.OrderCount != 0 || out.ValidOrderCount != 0 || out.Turnover != 0 || out.RefundAmount != 0 || out.UserCount != 0 {
		t.Fatalf("expected empty overview, got %+v", out)
	}
}

func TestMySQLReportRepo_QueryTrendFromTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLReportRepo(db)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 4, 3, 23, 59, 59, 0, time.Local)

	mock.ExpectQuery("SELECT DATE_FORMAT\\(order_time, '%Y-%m-%d'\\) AS time_key, COALESCE\\(SUM\\(amount\\), 0\\) AS v FROM orders_202604 WHERE order_time >= \\? AND order_time <= \\? AND status = \\? GROUP BY DATE_FORMAT\\(order_time, '%Y-%m-%d'\\) ORDER BY DATE_FORMAT\\(order_time, '%Y-%m-%d'\\) ASC").
		WithArgs(from, to, domain.OrderStatusCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"time_key", "v"}).AddRow("2026-04-01", 1000).AddRow("2026-04-02", 2000))

	points, err := repo.QueryTrendFromTable(context.Background(), "orders_202604", domain.TrendQuery{From: from, To: to, Granularity: "day"})
	if err != nil {
		t.Fatalf("QueryTrendFromTable failed: %v", err)
	}
	if len(points) != 2 || points[0].TimeKey != "2026-04-01" || points[1].Value != 2000 {
		t.Fatalf("unexpected points: %+v", points)
	}
}
