package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

func (s *Service) RunOnce(ctx context.Context, job domain.JobType) (*domain.RunSummary, error) {
	startAll := time.Now()
	if s.deps.Scanner == nil || s.deps.Executor == nil || s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "runOnce deps not initialized", nil)
	}
	if job == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty job type", nil)
	}

	unlock, locked, err := s.tryLock(ctx, "compensation:"+string(job))
	if err != nil {
		return nil, err
	}
	if !locked {
		return &domain.RunSummary{JobType: job, ElapsedMs: time.Since(startAll).Milliseconds()}, nil
	}
	defer func() { _ = unlock() }()

	items, err := s.deps.Scanner.Scan(ctx, job, s.deps.BatchSize)
	if err != nil {
		return nil, err
	}
	summary := &domain.RunSummary{JobType: job, Total: len(items), ScanCount: len(items)}
	for _, item := range items {
		s.runOne(ctx, item, summary)
	}
	s.summaryFinalize(ctx, summary, startAll)
	return summary, nil
}

func (s *Service) runOne(ctx context.Context, item domain.TaskItem, summary *domain.RunSummary) {
	started := time.Now()
	run := func(runCtx context.Context) error {
		return s.deps.Executor.Execute(runCtx, item)
	}
	var execErr error
	if s.deps.Tx != nil {
		// Tx boundary is at app layer, one task per transaction.
		execErr = s.deps.Tx.RunInTx(ctx, run)
	} else {
		execErr = run(ctx)
	}
	if execErr != nil {
		summary.Failed++
		_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{
			TaskID:     item.TaskID,
			JobType:    item.JobType,
			Status:     domain.TaskFailed,
			Reason:     execErr.Error(),
			RetryCount: item.RetryCount,
			StartedAt:  started,
			FinishedAt: time.Now(),
			DurationMs: time.Since(started).Milliseconds(),
		})
		_ = s.deps.Repo.MarkFailed(ctx, item.TaskID, execErr.Error())
		return
	}
	if doneErr := s.deps.Repo.MarkDone(ctx, item.TaskID); doneErr != nil {
		summary.Skipped++
		_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{
			TaskID:     item.TaskID,
			JobType:    item.JobType,
			Status:     domain.TaskSkipped,
			Reason:     doneErr.Error(),
			RetryCount: item.RetryCount,
			StartedAt:  started,
			FinishedAt: time.Now(),
			DurationMs: time.Since(started).Milliseconds(),
		})
		return
	}
	summary.Done++
	_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{
		TaskID:     item.TaskID,
		JobType:    item.JobType,
		Status:     domain.TaskDone,
		RetryCount: item.RetryCount,
		StartedAt:  started,
		FinishedAt: time.Now(),
		DurationMs: time.Since(started).Milliseconds(),
	})

	// Side effects are post-commit and best-effort.
	// TODO: use outbox/event bus for reliable post-task notifications.
	if s.deps.Cache != nil {
		_ = s.deps.Cache.Ping(ctx)
	}
	if s.deps.MQ != nil {
		_ = s.deps.MQ.Ping(ctx)
	}
	if s.deps.WebSocket != nil {
		_ = s.deps.WebSocket.Ping(ctx)
	}
}

func (s *Service) summaryFinalize(ctx context.Context, summary *domain.RunSummary, started time.Time) {
	if summary == nil {
		return
	}
	summary.SuccessCount = summary.Done
	summary.FailCount = summary.Failed
	summary.SkipCount = summary.Skipped
	summary.ElapsedMs = time.Since(started).Milliseconds()
	if s.deps.Metrics != nil {
		s.deps.Metrics.Observe(ctx, *summary)
	}
}

func (s *Service) tryLock(ctx context.Context, key string) (func() error, bool, error) {
	if s.deps.Lock == nil {
		return func() error { return nil }, true, nil
	}
	return s.deps.Lock.TryLock(ctx, key, s.deps.LockTTL)
}
