package outbox

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Relay struct {
	Repo      domain.OutboxRepository
	Publisher domain.EventPublisher
	BatchSize int
}

func NewRelay(repo domain.OutboxRepository, publisher domain.EventPublisher, batchSize int) *Relay {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Relay{Repo: repo, Publisher: publisher, BatchSize: batchSize}
}

func (r *Relay) Flush(ctx context.Context) error {
	if r.Repo == nil || r.Publisher == nil {
		return nil
	}
	pending, err := r.Repo.FetchPending(ctx, r.BatchSize)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, evt := range pending {
		if pubErr := r.Publisher.Publish(ctx, evt); pubErr != nil {
			nextRetry := now.Add(2 * time.Second)
			_ = r.Repo.MarkFailed(ctx, evt.EventID, pubErr.Error(), nextRetry)
			continue
		}
		_ = r.Repo.MarkPublished(ctx, evt.EventID, time.Now())
	}
	return nil
}
