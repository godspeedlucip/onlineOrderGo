package outbox

import (
	"context"
	"sync"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type item struct {
	evt         domain.OrderEvent
	publishedAt *time.Time
}

type InMemoryRepo struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{items: make(map[string]item)}
}

func (r *InMemoryRepo) Save(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	r.mu.Lock()
	r.items[evt.EventID] = item{evt: evt}
	r.mu.Unlock()
	return nil
}

func (r *InMemoryRepo) FetchPending(ctx context.Context, limit int) ([]domain.OrderEvent, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.OrderEvent, 0, limit)
	for _, it := range r.items {
		if it.publishedAt != nil {
			continue
		}
		out = append(out, it.evt)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *InMemoryRepo) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	_ = ctx
	r.mu.Lock()
	old, ok := r.items[eventID]
	if ok {
		old.publishedAt = &publishedAt
		r.items[eventID] = old
	}
	r.mu.Unlock()
	return nil
}
