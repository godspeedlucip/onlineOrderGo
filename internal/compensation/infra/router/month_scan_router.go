package router

import (
	"context"
	"fmt"
	"time"
)

type MonthScanRouter struct {
	baseTable       string
	shardingEnabled bool
	scanMonths      int
}

func NewMonthScanRouter(baseTable string, shardingEnabled bool, scanMonths int) *MonthScanRouter {
	if baseTable == "" {
		baseTable = "orders"
	}
	if scanMonths <= 0 {
		scanMonths = 2
	}
	return &MonthScanRouter{
		baseTable:       baseTable,
		shardingEnabled: shardingEnabled,
		scanMonths:      scanMonths,
	}
}

func (r *MonthScanRouter) CandidateTables(ctx context.Context, anchor time.Time) ([]string, error) {
	_ = ctx
	if r == nil {
		return nil, fmt.Errorf("router is not initialized")
	}
	if !r.shardingEnabled {
		return []string{r.baseTable}, nil
	}
	if anchor.IsZero() {
		anchor = time.Now()
	}
	out := make([]string, 0, r.scanMonths)
	for i := 0; i < r.scanMonths; i++ {
		t := anchor.AddDate(0, -i, 0)
		out = append(out, fmt.Sprintf("%s_%04d%02d", r.baseTable, t.Year(), int(t.Month())))
	}
	return out, nil
}
