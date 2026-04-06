package executor

import (
	"context"
	"encoding/json"
	"errors"

	"go-baseline-skeleton/internal/compensation/domain"
)

type CompositeExecutor struct{}

func NewCompositeExecutor() *CompositeExecutor { return &CompositeExecutor{} }

func (e *CompositeExecutor) Execute(ctx context.Context, item domain.TaskItem) error {
	_ = ctx
	if item.TaskID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty taskId", nil)
	}
	switch item.JobType {
	case domain.JobOrderTimeoutCancel:
		return e.execOrderTimeoutCancel(item)
	case domain.JobPaymentStateFix:
		return e.execPaymentStateFix(item)
	case domain.JobOutboxRetry:
		return e.execOutboxRetry(item)
	default:
		return domain.NewBizError(domain.CodeInvalidArgument, "unsupported job type", nil)
	}
}

func (e *CompositeExecutor) execOrderTimeoutCancel(item domain.TaskItem) error {
	_ = item
	// TODO: call order app usecase: cancel unpaid overdue order with state precondition.
	return nil
}

func (e *CompositeExecutor) execPaymentStateFix(item domain.TaskItem) error {
	if len(item.Payload) == 0 {
		// no payload means scanner-only fix path.
		return nil
	}
	var v map[string]any
	if err := json.Unmarshal(item.Payload, &v); err != nil {
		return errors.New("invalid payment fix payload")
	}
	// TODO: check payment/order status mismatch and perform conditional transition.
	return nil
}

func (e *CompositeExecutor) execOutboxRetry(item domain.TaskItem) error {
	_ = item
	// TODO: republish pending outbox message; ensure idempotent event key.
	return nil
}
