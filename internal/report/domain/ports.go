package domain

import (
	"context"
	"time"
)

type ReportUsecase interface {
	QueryOverview(ctx context.Context, q OverviewQuery) (*OverviewReport, error)
	QueryTrend(ctx context.Context, q TrendQuery) (*TrendReport, error)
	QueryOrderList(ctx context.Context, q OrderListQuery) (*OrderListResult, error)
}

type ShardRouter interface {
	ResolveTables(ctx context.Context, from, to time.Time) ([]string, error)
	ResolveWriteTable(ctx context.Context, at time.Time) (string, error)
}

type ReportRepository interface {
	QueryOverviewFromTable(ctx context.Context, table string, q OverviewQuery) (*OverviewPartial, error)
	QueryTrendFromTable(ctx context.Context, table string, q TrendQuery) ([]TrendPoint, error)
	QueryOrdersFromTable(ctx context.Context, table string, q OrderListQuery) ([]OrderRow, error)
}

type ReportCache interface {
	Get(ctx context.Context, key string, out any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

type ShardTableManager interface {
	EnsureTable(ctx context.Context, table string) error
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type CachePort interface {
	Ping(ctx context.Context) error
}

type MQPort interface {
	Ping(ctx context.Context) error
}

type WebSocketPort interface {
	Ping(ctx context.Context) error
}

type PaymentPort interface {
	Ping(ctx context.Context) error
}