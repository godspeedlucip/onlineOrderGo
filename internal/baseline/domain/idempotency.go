package domain

import "time"

type IdempotencyStatus string

const (
	StatusProcessing IdempotencyStatus = "PROCESSING"
	StatusSucceeded  IdempotencyStatus = "SUCCEEDED"
	StatusFailed     IdempotencyStatus = "FAILED"
)

type IdempotencyRecord struct {
	Scene      string
	Key        string
	Token      string
	Status     IdempotencyStatus
	Payload    []byte
	Reason     string
	UpdatedAt  time.Time
	ExpireAt   time.Time
}

func (r *IdempotencyRecord) CanTransitionTo(next IdempotencyStatus) bool {
	switch r.Status {
	case StatusProcessing:
		return next == StatusSucceeded || next == StatusFailed
	case StatusFailed:
		// FAILED -> PROCESSING is handled by Acquire with a new token.
		return false
	case StatusSucceeded:
		return false
	default:
		return false
	}
}