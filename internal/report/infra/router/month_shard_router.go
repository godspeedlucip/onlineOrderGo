package router

import (
	"context"
	"fmt"
	"time"
)

type MonthShardRouter struct {
	baseTable string
}

func NewMonthShardRouter(baseTable string) *MonthShardRouter {
	if baseTable == "" {
		baseTable = "orders"
	}
	return &MonthShardRouter{baseTable: baseTable}
}

func (r *MonthShardRouter) ResolveTables(ctx context.Context, from, to time.Time) ([]string, error) {
	_ = ctx
	if to.Before(from) {
		return nil, fmt.Errorf("invalid range")
	}

	cursor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, to.Location())
	tables := make([]string, 0)
	for !cursor.After(end) {
		tables = append(tables, fmt.Sprintf("%s_%04d%02d", r.baseTable, cursor.Year(), int(cursor.Month())))
		cursor = cursor.AddDate(0, 1, 0)
	}
	return tables, nil
}

func (r *MonthShardRouter) ResolveWriteTable(ctx context.Context, at time.Time) (string, error) {
	_ = ctx
	if at.IsZero() {
		at = time.Now()
	}
	return fmt.Sprintf("%s_%04d%02d", r.baseTable, at.Year(), int(at.Month())), nil
}