package metrics

import (
	"context"
	"sync"

	"go-baseline-skeleton/internal/compensation/domain"
)

type InMemoryRecorder struct {
	mu      sync.Mutex
	records []domain.RunSummary
}

func NewInMemoryRecorder() *InMemoryRecorder {
	return &InMemoryRecorder{records: make([]domain.RunSummary, 0, 16)}
}

func (r *InMemoryRecorder) Observe(ctx context.Context, summary domain.RunSummary) {
	_ = ctx
	r.mu.Lock()
	r.records = append(r.records, summary)
	r.mu.Unlock()
}

func (r *InMemoryRecorder) Snapshot() []domain.RunSummary {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RunSummary, len(r.records))
	copy(out, r.records)
	return out
}
