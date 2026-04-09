package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

func (s *Service) ReplayFailed(ctx context.Context, limit int) (*domain.RunSummary, error) {
	startAll := time.Now()
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
		return &domain.RunSummary{JobType: domain.JobReplayFailed, ElapsedMs: time.Since(startAll).Milliseconds()}, nil
	}
	defer func() { _ = unlock() }()

	items, err := s.deps.Scanner.ScanFailed(ctx, limit)
	if err != nil {
		return nil, err
	}
	summary := &domain.RunSummary{JobType: domain.JobReplayFailed, Total: len(items), ScanCount: len(items)}
	for _, item := range items {
		s.runOne(ctx, item, summary)
	}
	s.summaryFinalize(ctx, summary, startAll)
	return summary, nil
}
