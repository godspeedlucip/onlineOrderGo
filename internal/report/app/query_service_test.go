package app

import (
	"context"
	"testing"
	"time"

	"go-baseline-skeleton/internal/report/domain"
)

type fakeReportRepo struct {
	overviewByTable map[string]*domain.OverviewPartial
	trendByTable    map[string][]domain.TrendPoint
	ordersByTable   map[string][]domain.OrderRow
}

func (r *fakeReportRepo) QueryOverviewFromTable(ctx context.Context, table string, q domain.OverviewQuery) (*domain.OverviewPartial, error) {
	_ = ctx
	_ = q
	if v, ok := r.overviewByTable[table]; ok {
		return v, nil
	}
	return &domain.OverviewPartial{}, nil
}

func (r *fakeReportRepo) QueryTrendFromTable(ctx context.Context, table string, q domain.TrendQuery) ([]domain.TrendPoint, error) {
	_ = ctx
	_ = q
	if v, ok := r.trendByTable[table]; ok {
		return v, nil
	}
	return []domain.TrendPoint{}, nil
}

func (r *fakeReportRepo) QueryOrdersFromTable(ctx context.Context, table string, q domain.OrderListQuery) ([]domain.OrderRow, error) {
	_ = ctx
	_ = q
	if v, ok := r.ordersByTable[table]; ok {
		return v, nil
	}
	return []domain.OrderRow{}, nil
}

type fakeRouter struct {
	tables    []string
	writeTable string
}

func (r *fakeRouter) ResolveTables(ctx context.Context, from, to time.Time) ([]string, error) {
	_ = ctx
	_ = from
	_ = to
	return append([]string(nil), r.tables...), nil
}

func (r *fakeRouter) ResolveWriteTable(ctx context.Context, at time.Time) (string, error) {
	_ = ctx
	_ = at
	return r.writeTable, nil
}

type fakeDDL struct {
	lastTable string
}

func (d *fakeDDL) EnsureTable(ctx context.Context, table string) error {
	_ = ctx
	d.lastTable = table
	return nil
}

func TestService_QueryOverview_CrossMonthAndEmptyMonth(t *testing.T) {
	svc := NewService(Deps{
		Repo: &fakeReportRepo{
			overviewByTable: map[string]*domain.OverviewPartial{
				"orders_202603": {OrderCount: 2, ValidOrderCount: 1, Turnover: 1000, RefundAmount: 100, UserCount: 2},
				"orders_202604": {OrderCount: 3, ValidOrderCount: 2, Turnover: 3000, RefundAmount: 0, UserCount: 2},
				// orders_202605 intentionally empty
			},
		},
		Router: &fakeRouter{tables: []string{"orders_202603", "orders_202604", "orders_202605"}},
	})

	out, err := svc.QueryOverview(context.Background(), domain.OverviewQuery{
		From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		To:   time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("QueryOverview failed: %v", err)
	}
	if out.OrderCount != 5 || out.ValidOrderCount != 3 || out.Turnover != 4000 || out.RefundAmount != 100 || out.UserCount != 4 {
		t.Fatalf("unexpected overview aggregate: %+v", out)
	}
}

func TestService_QueryOrderList_CrossMonthHistorySorted(t *testing.T) {
	svc := NewService(Deps{
		Repo: &fakeReportRepo{
			ordersByTable: map[string][]domain.OrderRow{
				"orders_202603": {
					{OrderID: 11, OrderNumber: "N11", CreatedAt: time.Date(2026, 3, 31, 23, 0, 0, 0, time.Local), Amount: 100},
				},
				"orders_202604": {
					{OrderID: 21, OrderNumber: "N21", CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local), Amount: 200},
					{OrderID: 22, OrderNumber: "N22", CreatedAt: time.Date(2026, 4, 2, 10, 0, 0, 0, time.Local), Amount: 300},
				},
			},
		},
		Router: &fakeRouter{tables: []string{"orders_202603", "orders_202604"}},
	})

	out, err := svc.QueryOrderList(context.Background(), domain.OrderListQuery{
		From:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		To:       time.Date(2026, 4, 30, 23, 59, 59, 0, time.Local),
		Page:     1,
		PageSize: 2,
		Desc:     true,
	})
	if err != nil {
		t.Fatalf("QueryOrderList failed: %v", err)
	}
	if out.Total != 3 || len(out.List) != 2 || out.List[0].OrderID != 22 || out.List[1].OrderID != 21 {
		t.Fatalf("unexpected list aggregate: %+v", out)
	}
}

func TestService_QueryTrend_MergeByTimeKey(t *testing.T) {
	svc := NewService(Deps{
		Repo: &fakeReportRepo{
			trendByTable: map[string][]domain.TrendPoint{
				"orders_202603": {{TimeKey: "2026-03-31", Value: 100}},
				"orders_202604": {{TimeKey: "2026-04-01", Value: 200}, {TimeKey: "2026-04-02", Value: 300}},
			},
		},
		Router: &fakeRouter{tables: []string{"orders_202603", "orders_202604"}},
	})

	out, err := svc.QueryTrend(context.Background(), domain.TrendQuery{
		From:        time.Date(2026, 3, 31, 0, 0, 0, 0, time.Local),
		To:          time.Date(2026, 4, 2, 23, 59, 59, 0, time.Local),
		Granularity: "day",
	})
	if err != nil {
		t.Fatalf("QueryTrend failed: %v", err)
	}
	if len(out.Series) != 3 || out.Series[0].TimeKey != "2026-03-31" || out.Series[2].Value != 300 {
		t.Fatalf("unexpected trend: %+v", out.Series)
	}
}

func TestService_PrepareWriteShard_Boundary(t *testing.T) {
	ddl := &fakeDDL{}
	svc := NewService(Deps{
		Repo:   &fakeReportRepo{},
		Router: &fakeRouter{writeTable: "orders_202604"},
		DDL:    ddl,
	})
	table, err := svc.PrepareWriteShard(context.Background(), time.Date(2026, 4, 8, 10, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("PrepareWriteShard failed: %v", err)
	}
	if table != "orders_202604" || ddl.lastTable != "orders_202604" {
		t.Fatalf("unexpected write boundary result table=%s ddl=%s", table, ddl.lastTable)
	}
}
