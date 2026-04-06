package ddl

import "context"

type ShardTableManager struct{}

func NewShardTableManager() *ShardTableManager {
	return &ShardTableManager{}
}

func (m *ShardTableManager) EnsureTable(ctx context.Context, table string) error {
	_ = ctx
	_ = table
	// TODO: implement DDL create-if-not-exists for write phase.
	return nil
}