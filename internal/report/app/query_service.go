package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go-baseline-skeleton/internal/report/domain"
)

type Deps struct {
	Repo    domain.ReportRepository
	Router  domain.ShardRouter
	Cache   domain.ReportCache
	Tx      domain.TxManager
	DDL     domain.ShardTableManager

	// Optional cross-domain dependencies. Keep injected for future expansion.
	CacheInfra domain.CachePort
	MQ         domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	OverviewTTL time.Duration
	TrendTTL    time.Duration
	ListTTL     time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.OverviewTTL <= 0 {
		deps.OverviewTTL = 2 * time.Minute
	}
	if deps.TrendTTL <= 0 {
		deps.TrendTTL = 2 * time.Minute
	}
	if deps.ListTTL <= 0 {
		deps.ListTTL = 30 * time.Second
	}
	return &Service{deps: deps}
}

func (s *Service) QueryOverview(ctx context.Context, q domain.OverviewQuery) (*domain.OverviewReport, error) {
	if err := validateRange(q.From, q.To); err != nil {
		return nil, err
	}
	if s.deps.Repo == nil || s.deps.Router == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "report dependencies not initialized", nil)
	}

	cacheKey := fmt.Sprintf("report:overview:from=%d:to=%d:store=%d", q.From.Unix(), q.To.Unix(), q.StoreID)
	if s.deps.Cache != nil {
		var cached domain.OverviewReport
		if ok, err := s.deps.Cache.Get(ctx, cacheKey, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	tables, err := s.deps.Router.ResolveTables(ctx, q.From, q.To)
	if err != nil {
		return nil, err
	}
	if len(tables) == 0 {
		return &domain.OverviewReport{}, nil
	}

	out := &domain.OverviewReport{}
	for _, table := range tables {
		part, qErr := s.deps.Repo.QueryOverviewFromTable(ctx, table, q)
		if qErr != nil {
			return nil, qErr
		}
		if part == nil {
			continue
		}
		out.OrderCount += part.OrderCount
		out.ValidOrderCount += part.ValidOrderCount
		out.Turnover += part.Turnover
		out.RefundAmount += part.RefundAmount
		out.UserCount += part.UserCount
	}

	if s.deps.Cache != nil {
		_ = s.deps.Cache.Set(ctx, cacheKey, out, s.deps.OverviewTTL)
	}
	return out, nil
}

func (s *Service) QueryTrend(ctx context.Context, q domain.TrendQuery) (*domain.TrendReport, error) {
	if err := validateRange(q.From, q.To); err != nil {
		return nil, err
	}
	if s.deps.Repo == nil || s.deps.Router == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "report dependencies not initialized", nil)
	}

	cacheKey := fmt.Sprintf("report:trend:from=%d:to=%d:g=%s:store=%d", q.From.Unix(), q.To.Unix(), q.Granularity, q.StoreID)
	if s.deps.Cache != nil {
		var cached domain.TrendReport
		if ok, err := s.deps.Cache.Get(ctx, cacheKey, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	tables, err := s.deps.Router.ResolveTables(ctx, q.From, q.To)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]int64)
	for _, table := range tables {
		points, qErr := s.deps.Repo.QueryTrendFromTable(ctx, table, q)
		if qErr != nil {
			return nil, qErr
		}
		for _, p := range points {
			merged[p.TimeKey] += p.Value
		}
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	series := make([]domain.TrendPoint, 0, len(keys))
	for _, k := range keys {
		series = append(series, domain.TrendPoint{TimeKey: k, Value: merged[k]})
	}

	out := &domain.TrendReport{Series: series}
	if s.deps.Cache != nil {
		_ = s.deps.Cache.Set(ctx, cacheKey, out, s.deps.TrendTTL)
	}
	return out, nil
}

func (s *Service) QueryOrderList(ctx context.Context, q domain.OrderListQuery) (*domain.OrderListResult, error) {
	if err := validateRange(q.From, q.To); err != nil {
		return nil, err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	if s.deps.Repo == nil || s.deps.Router == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "report dependencies not initialized", nil)
	}

	cacheKey := fmt.Sprintf("report:orders:from=%d:to=%d:store=%d:p=%d:s=%d:sort=%s:desc=%t", q.From.Unix(), q.To.Unix(), q.StoreID, q.Page, q.PageSize, q.SortBy, q.Desc)
	if s.deps.Cache != nil {
		var cached domain.OrderListResult
		if ok, err := s.deps.Cache.Get(ctx, cacheKey, &cached); err == nil && ok {
			return &cached, nil
		}
	}

	tables, err := s.deps.Router.ResolveTables(ctx, q.From, q.To)
	if err != nil {
		return nil, err
	}

	all := make([]domain.OrderRow, 0)
	for _, table := range tables {
		rows, qErr := s.deps.Repo.QueryOrdersFromTable(ctx, table, q)
		if qErr != nil {
			return nil, qErr
		}
		all = append(all, rows...)
	}

	sort.SliceStable(all, func(i, j int) bool {
		if q.Desc {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})

	total := int64(len(all))
	start := (q.Page - 1) * q.PageSize
	if start >= len(all) {
		out := &domain.OrderListResult{Total: total, List: []domain.OrderRow{}}
		if s.deps.Cache != nil {
			_ = s.deps.Cache.Set(ctx, cacheKey, out, s.deps.ListTTL)
		}
		return out, nil
	}
	end := start + q.PageSize
	if end > len(all) {
		end = len(all)
	}
	out := &domain.OrderListResult{Total: total, List: all[start:end]}
	if s.deps.Cache != nil {
		_ = s.deps.Cache.Set(ctx, cacheKey, out, s.deps.ListTTL)
	}
	return out, nil
}

// PrepareWriteShard binds shard table creation to app orchestration boundary.
// Write path can call this first, then continue write operations in the same app flow.
func (s *Service) PrepareWriteShard(ctx context.Context, at time.Time) (string, error) {
	if s.deps.Router == nil {
		return "", domain.NewBizError(domain.CodeInternal, "shard router not initialized", nil)
	}
	table, err := s.deps.Router.ResolveWriteTable(ctx, at)
	if err != nil {
		return "", err
	}
	if s.deps.DDL == nil {
		return table, nil
	}
	ensureFn := func(runCtx context.Context) error {
		return s.deps.DDL.EnsureTable(runCtx, table)
	}
	if s.deps.Tx != nil {
		if txErr := s.deps.Tx.RunInTx(ctx, ensureFn); txErr != nil {
			return "", txErr
		}
		return table, nil
	}
	if err := ensureFn(ctx); err != nil {
		return "", err
	}
	return table, nil
}

func validateRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() {
		return domain.NewBizError(domain.CodeInvalidArgument, "from/to is required", nil)
	}
	if to.Before(from) {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid time range", nil)
	}
	return nil
}

// Read chain usually does not require explicit tx.
// TODO: for write phase (e.g. shard DDL + write routing), use Tx in app layer to orchestrate steps.
