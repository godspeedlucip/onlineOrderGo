package tx

import "context"

type NoopManager struct{}

func NewNoopManager() *NoopManager { return &NoopManager{} }

func (m *NoopManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
