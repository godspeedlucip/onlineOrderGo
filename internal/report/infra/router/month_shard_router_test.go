package router

import (
	"context"
	"testing"
	"time"
)

func TestMonthShardRouter_ResolveTablesCrossMonth(t *testing.T) {
	r := NewMonthShardRouterWithOptions("orders", true, 0)
	from := time.Date(2026, 3, 31, 10, 0, 0, 0, time.Local)
	to := time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local)

	tables, err := r.ResolveTables(context.Background(), from, to)
	if err != nil {
		t.Fatalf("ResolveTables failed: %v", err)
	}
	want := []string{"orders_202603", "orders_202604", "orders_202605"}
	if len(tables) != len(want) {
		t.Fatalf("unexpected table count: got=%v want=%v", tables, want)
	}
	for i := range want {
		if tables[i] != want[i] {
			t.Fatalf("unexpected table[%d]=%s want=%s", i, tables[i], want[i])
		}
	}
}

func TestMonthShardRouter_ResolveTablesNonSharding(t *testing.T) {
	r := NewMonthShardRouterWithOptions("orders", false, 0)
	tables, err := r.ResolveTables(context.Background(), time.Now().AddDate(0, 0, -3), time.Now())
	if err != nil {
		t.Fatalf("ResolveTables failed: %v", err)
	}
	if len(tables) != 1 || tables[0] != "orders" {
		t.Fatalf("unexpected non-sharding tables: %+v", tables)
	}
}

func TestMonthShardRouter_ResolveTablesScanOverflow(t *testing.T) {
	r := NewMonthShardRouterWithOptions("orders", true, 1)
	_, err := r.ResolveTables(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local))
	if err == nil {
		t.Fatal("expected scan overflow error")
	}
}
