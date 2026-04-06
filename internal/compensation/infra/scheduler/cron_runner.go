package scheduler

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type CronRunner struct {
	Usecase  domain.CompensationUsecase
	Interval time.Duration
	Job      domain.JobType
}

func NewCronRunner(usecase domain.CompensationUsecase, job domain.JobType, interval time.Duration) *CronRunner {
	if interval <= 0 {
		interval = time.Minute
	}
	return &CronRunner{Usecase: usecase, Job: job, Interval: interval}
}

func (r *CronRunner) Start(ctx context.Context) error {
	if r.Usecase == nil {
		return nil
	}
	if r.Job == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty scheduled job type", nil)
	}
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _ = r.Usecase.RunOnce(ctx, r.Job)
		}
	}
}
