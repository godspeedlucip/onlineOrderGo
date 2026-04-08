package router

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type MonthShardRouter struct {
	baseTable       string
	shardingEnabled bool
	maxScanMonths   int
}

func NewMonthShardRouter(baseTable string) *MonthShardRouter {
	return NewMonthShardRouterWithOptions(baseTable, true, 0)
}

func NewMonthShardRouterWithOptions(baseTable string, shardingEnabled bool, maxScanMonths int) *MonthShardRouter {
	if baseTable == "" {
		baseTable = "orders"
	}
	return &MonthShardRouter{
		baseTable:       baseTable,
		shardingEnabled: shardingEnabled,
		maxScanMonths:   maxScanMonths,
	}
}

func (r *MonthShardRouter) ResolveTables(ctx context.Context, from, to time.Time) ([]string, error) {
	_ = ctx
	if r == nil {
		return nil, fmt.Errorf("router is not initialized")
	}
	if !r.shardingEnabled {
		return []string{r.baseTable}, nil
	}
	if to.Before(from) {
		return nil, fmt.Errorf("invalid range")
	}

	startMonth := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	endMonth := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, to.Location())

	tables := make([]string, 0)
	for cursor := startMonth; !cursor.After(endMonth); cursor = cursor.AddDate(0, 1, 0) {
		tables = append(tables, fmt.Sprintf("%s_%04d%02d", r.baseTable, cursor.Year(), int(cursor.Month())))
		if r.maxScanMonths > 0 && len(tables) > r.maxScanMonths {
			return nil, fmt.Errorf("scan months overflow: max=%d", r.maxScanMonths)
		}
	}
	sort.Strings(tables)
	return tables, nil
}

func (r *MonthShardRouter) ResolveWriteTable(ctx context.Context, at time.Time) (string, error) {
	_ = ctx
	if r == nil {
		return "", fmt.Errorf("router is not initialized")
	}
	if !r.shardingEnabled {
		return r.baseTable, nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	return fmt.Sprintf("%s_%04d%02d", r.baseTable, at.Year(), int(at.Month())), nil
}
