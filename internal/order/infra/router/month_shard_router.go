package router

import (
	"context"
	"fmt"
	"time"
)

type MonthShardRouter struct {
	baseTable       string
	shardingEnabled bool
}

func NewMonthShardRouter(baseTable string, shardingEnabled bool) *MonthShardRouter {
	if baseTable == "" {
		baseTable = "orders"
	}
	return &MonthShardRouter{baseTable: baseTable, shardingEnabled: shardingEnabled}
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
