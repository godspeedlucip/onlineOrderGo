package worker

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/order_event/domain"
)

type Runner struct {
	PublishUsecase domain.PublishUsecase
	ConsumeUsecase domain.ConsumeUsecase
}

func NewRunner(pub domain.PublishUsecase, consume domain.ConsumeUsecase) *Runner {
	return &Runner{PublishUsecase: pub, ConsumeUsecase: consume}
}

func (r *Runner) Start(ctx context.Context) error {
	if starter, ok := r.ConsumeUsecase.(interface{ StartConsume(context.Context) error }); ok {
		go func() {
			_ = starter.StartConsume(ctx)
		}()
	}
	if flusher, ok := r.PublishUsecase.(interface{ FlushOutbox(context.Context) error }); ok {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = flusher.FlushOutbox(ctx)
				}
			}
		}()
	}
	<-ctx.Done()
	return ctx.Err()
}
