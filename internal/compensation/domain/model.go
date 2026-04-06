package domain

import "time"

type JobType string

const (
	JobOrderTimeoutCancel JobType = "ORDER_TIMEOUT_CANCEL"
	JobPaymentStateFix    JobType = "PAYMENT_STATE_FIX"
	JobOutboxRetry        JobType = "OUTBOX_RETRY"
	JobReplayFailed       JobType = "REPLAY_FAILED"
)

type TaskStatus string

const (
	TaskPending TaskStatus = "PENDING"
	TaskDone    TaskStatus = "DONE"
	TaskFailed  TaskStatus = "FAILED"
	TaskSkipped TaskStatus = "SKIPPED"
)

type TaskItem struct {
	TaskID      string
	JobType     JobType
	BizKey      string
	Payload     []byte
	ScheduledAt time.Time
}

type TaskRunRecord struct {
	TaskID      string
	JobType     JobType
	Status      TaskStatus
	Reason      string
	StartedAt   time.Time
	FinishedAt  time.Time
}

type RunSummary struct {
	JobType JobType `json:"jobType"`
	Total   int     `json:"total"`
	Done    int     `json:"done"`
	Failed  int     `json:"failed"`
	Skipped int     `json:"skipped"`
}
