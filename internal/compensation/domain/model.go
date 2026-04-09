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
	TaskRunning TaskStatus = "RUNNING"
	TaskDone    TaskStatus = "DONE"
	TaskFailed  TaskStatus = "FAILED"
	TaskDead    TaskStatus = "DEAD"
	TaskSkipped TaskStatus = "SKIPPED"
)

type TaskItem struct {
	TaskID         string
	JobType        JobType
	BizKey         string
	Payload        []byte
	ScheduledAt    time.Time
	RetryCount     int
	MaxRetry       int
	NextExecuteAt  time.Time
	DeadReason     string
	ShardTable     string
	ClaimToken     string
}

type TaskRunRecord struct {
	TaskID      string
	JobType     JobType
	Status      TaskStatus
	Reason      string
	RetryCount  int
	StartedAt   time.Time
	FinishedAt  time.Time
	DurationMs  int64
}

type RunSummary struct {
	JobType     JobType `json:"jobType"`
	Total       int     `json:"total"`
	Done        int     `json:"done"`
	Failed      int     `json:"failed"`
	Skipped     int     `json:"skipped"`
	ScanCount   int     `json:"scanCount"`
	SuccessCount int    `json:"successCount"`
	FailCount   int     `json:"failCount"`
	SkipCount   int     `json:"skipCount"`
	ElapsedMs   int64   `json:"elapsedMs"`
}
