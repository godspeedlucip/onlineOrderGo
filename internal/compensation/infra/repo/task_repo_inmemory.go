package repo

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type store struct {
	mu     sync.Mutex
	tasks  map[string]domain.TaskItem
	state  map[string]domain.TaskStatus
	reason map[string]string
	runs   []domain.TaskRunRecord
}

var globalStore = newStore()

func newStore() *store {
	now := time.Now()
	tasks := map[string]domain.TaskItem{
		"task_order_timeout_1": {TaskID: "task_order_timeout_1", JobType: domain.JobOrderTimeoutCancel, BizKey: "order:1001", ScheduledAt: now.Add(-5 * time.Minute)},
		"task_payment_fix_1":   {TaskID: "task_payment_fix_1", JobType: domain.JobPaymentStateFix, BizKey: "order:1002", ScheduledAt: now.Add(-2 * time.Minute)},
		"task_outbox_retry_1":  {TaskID: "task_outbox_retry_1", JobType: domain.JobOutboxRetry, BizKey: "outbox:5001", ScheduledAt: now.Add(-1 * time.Minute)},
	}
	state := map[string]domain.TaskStatus{
		"task_order_timeout_1": domain.TaskPending,
		"task_payment_fix_1":   domain.TaskFailed,
		"task_outbox_retry_1":  domain.TaskPending,
	}
	return &store{tasks: tasks, state: state, reason: make(map[string]string), runs: make([]domain.TaskRunRecord, 0)}
}

type InMemoryTaskRepo struct{}

func NewInMemoryTaskRepo() *InMemoryTaskRepo { return &InMemoryTaskRepo{} }

func (r *InMemoryTaskRepo) SaveRun(ctx context.Context, rec domain.TaskRunRecord) error {
	_ = ctx
	globalStore.mu.Lock()
	globalStore.runs = append(globalStore.runs, rec)
	globalStore.mu.Unlock()
	return nil
}

func (r *InMemoryTaskRepo) MarkDone(ctx context.Context, taskID string) error {
	_ = ctx
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	st, ok := globalStore.state[taskID]
	if !ok {
		return domain.NewBizError(domain.CodeInvalidArgument, "task not found", nil)
	}
	if st == domain.TaskDone {
		return domain.NewBizError(domain.CodeConflict, "task already done", nil)
	}
	globalStore.state[taskID] = domain.TaskDone
	delete(globalStore.reason, taskID)
	return nil
}

func (r *InMemoryTaskRepo) MarkFailed(ctx context.Context, taskID, reason string) error {
	_ = ctx
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	if _, ok := globalStore.state[taskID]; !ok {
		return domain.NewBizError(domain.CodeInvalidArgument, "task not found", nil)
	}
	globalStore.state[taskID] = domain.TaskFailed
	globalStore.reason[taskID] = reason
	return nil
}

func scanByStatusAndJob(job domain.JobType, statuses map[domain.TaskStatus]struct{}, limit int, dueOnly bool) []domain.TaskItem {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now()
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	ids := make([]string, 0, len(globalStore.tasks))
	for id := range globalStore.tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]domain.TaskItem, 0, limit)
	for _, id := range ids {
		item := globalStore.tasks[id]
		if job != "" && item.JobType != job {
			continue
		}
		if _, ok := statuses[globalStore.state[id]]; !ok {
			continue
		}
		if dueOnly && item.ScheduledAt.After(now) {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func debugDump() string {
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	return fmt.Sprintf("tasks=%d runs=%d", len(globalStore.tasks), len(globalStore.runs))
}
