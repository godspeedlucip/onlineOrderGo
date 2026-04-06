package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

func (s *Service) ReplayFailed(ctx context.Context, limit int) (*domain.RunSummary, error) {
	if s.deps.Scanner == nil || s.deps.Executor == nil || s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "replay deps not initialized", nil)
	}
	if limit <= 0 {
		limit = s.deps.BatchSize
	}

	unlock, locked, err := s.tryLock(ctx, "compensation:replay_failed")
	if err != nil {
		return nil, err
	}
	if !locked {
		return &domain.RunSummary{JobType: domain.JobReplayFailed}, nil
	}
	defer func() { _ = unlock() }()

	items, err := s.deps.Scanner.ScanFailed(ctx, limit)
	if err != nil {
		return nil, err
	}
	summary := &domain.RunSummary{JobType: domain.JobReplayFailed, Total: len(items)}
	for _, item := range items {
		started := time.Now()
		run := func(runCtx context.Context) error {
			return s.deps.Executor.Execute(runCtx, item)
		}
		var execErr error
		if s.deps.Tx != nil {
			execErr = s.deps.Tx.RunInTx(ctx, run)
		} else {
			execErr = run(ctx)
		}
		if execErr != nil {
			summary.Failed++
			_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{TaskID: item.TaskID, JobType: item.JobType, Status: domain.TaskFailed, Reason: execErr.Error(), StartedAt: started, FinishedAt: time.Now()})
			_ = s.deps.Repo.MarkFailed(ctx, item.TaskID, execErr.Error())
			continue
		}
		if doneErr := s.deps.Repo.MarkDone(ctx, item.TaskID); doneErr != nil {
			summary.Skipped++
			_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{TaskID: item.TaskID, JobType: item.JobType, Status: domain.TaskSkipped, Reason: doneErr.Error(), StartedAt: started, FinishedAt: time.Now()})
			continue
		}
		summary.Done++
		_ = s.deps.Repo.SaveRun(ctx, domain.TaskRunRecord{TaskID: item.TaskID, JobType: item.JobType, Status: domain.TaskDone, StartedAt: started, FinishedAt: time.Now()})
	}
	return summary, nil
}
