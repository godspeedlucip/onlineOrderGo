package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type fakeScanner struct {
	items []domain.TaskItem
}

func (f *fakeScanner) Scan(ctx context.Context, job domain.JobType, limit int) ([]domain.TaskItem, error) {
	_ = ctx
	_ = job
	_ = limit
	return f.items, nil
}

func (f *fakeScanner) ScanFailed(ctx context.Context, limit int) ([]domain.TaskItem, error) {
	_ = ctx
	_ = limit
	return nil, nil
}

type fakeExecutor struct{}

func (f *fakeExecutor) Execute(ctx context.Context, item domain.TaskItem) error {
	_ = ctx
	if item.TaskID == "fail" {
		return errors.New("exec failed")
	}
	return nil
}

type fakeRepo struct {
	doneCalls   int
	failedCalls int
	runs        []domain.TaskRunRecord
}

func (f *fakeRepo) SaveRun(ctx context.Context, rec domain.TaskRunRecord) error {
	_ = ctx
	f.runs = append(f.runs, rec)
	return nil
}

func (f *fakeRepo) MarkDone(ctx context.Context, taskID string) error {
	_ = ctx
	_ = taskID
	f.doneCalls++
	return nil
}

func (f *fakeRepo) MarkFailed(ctx context.Context, taskID, reason string) error {
	_ = ctx
	_ = taskID
	_ = reason
	f.failedCalls++
	return nil
}

type fakeMetrics struct {
	last domain.RunSummary
	hit  int
}

func (f *fakeMetrics) Observe(ctx context.Context, summary domain.RunSummary) {
	_ = ctx
	f.hit++
	f.last = summary
}

func TestService_RunOnce_MetricsSummary(t *testing.T) {
	scanner := &fakeScanner{
		items: []domain.TaskItem{
			{TaskID: "ok", JobType: domain.JobOrderTimeoutCancel, ScheduledAt: time.Now().Add(-time.Minute)},
			{TaskID: "fail", JobType: domain.JobOrderTimeoutCancel, ScheduledAt: time.Now().Add(-time.Minute), RetryCount: 1},
		},
	}
	repo := &fakeRepo{}
	metrics := &fakeMetrics{}
	svc := NewService(Deps{
		Scanner:  scanner,
		Executor: &fakeExecutor{},
		Repo:     repo,
		Metrics:  metrics,
	})

	summary, err := svc.RunOnce(context.Background(), domain.JobOrderTimeoutCancel)
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	if summary.ScanCount != 2 || summary.Done != 1 || summary.Failed != 1 || summary.SuccessCount != 1 || summary.FailCount != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.ElapsedMs < 0 {
		t.Fatalf("invalid elapsed: %d", summary.ElapsedMs)
	}
	if metrics.hit != 1 || metrics.last.ScanCount != 2 {
		t.Fatalf("metrics not observed: hit=%d last=%+v", metrics.hit, metrics.last)
	}
	if repo.doneCalls != 1 || repo.failedCalls != 1 || len(repo.runs) != 2 {
		t.Fatalf("unexpected repo calls done=%d failed=%d runs=%d", repo.doneCalls, repo.failedCalls, len(repo.runs))
	}
}
