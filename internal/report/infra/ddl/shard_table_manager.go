package ddl

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var safeTableNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type ShardTableManager struct {
	db        *sql.DB
	baseTable string
}

func NewShardTableManager(db *sql.DB, baseTable string) *ShardTableManager {
	base := strings.TrimSpace(baseTable)
	if base == "" {
		base = "orders"
	}
	return &ShardTableManager{db: db, baseTable: base}
}

func (m *ShardTableManager) EnsureTable(ctx context.Context, table string) error {
	if m == nil || m.db == nil {
		// Read-first phase: allow empty manager in environments without write DB setup.
		return nil
	}
	target := strings.TrimSpace(table)
	if !safeTableNamePattern.MatchString(target) || !safeTableNamePattern.MatchString(m.baseTable) {
		return fmt.Errorf("invalid shard table name")
	}
	_, err := m.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s LIKE %s", target, m.baseTable))
	return err
}
