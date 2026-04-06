package repo

import (
	"context"

	"go-baseline-skeleton/internal/compensation/domain"
)

type InMemoryScanner struct{}

func NewInMemoryScanner() *InMemoryScanner { return &InMemoryScanner{} }

func (s *InMemoryScanner) Scan(ctx context.Context, job domain.JobType, limit int) ([]domain.TaskItem, error) {
	_ = ctx
	if job == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty job type", nil)
	}
	statuses := map[domain.TaskStatus]struct{}{domain.TaskPending: {}, domain.TaskFailed: {}}
	return scanByStatusAndJob(job, statuses, limit, true), nil
}

func (s *InMemoryScanner) ScanFailed(ctx context.Context, limit int) ([]domain.TaskItem, error) {
	_ = ctx
	statuses := map[domain.TaskStatus]struct{}{domain.TaskFailed: {}}
	return scanByStatusAndJob("", statuses, limit, true), nil
}
