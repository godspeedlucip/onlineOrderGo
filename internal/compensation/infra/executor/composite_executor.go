package executor

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
	orderdomain "go-baseline-skeleton/internal/order/domain"
)

type OutboxFlusher interface {
	FlushOutbox(ctx context.Context) error
}

type PaymentFixer interface {
	RepairPaymentState(ctx context.Context, item domain.TaskItem) error
}

type CompositeExecutor struct {
	order   orderdomain.OrderCommandUsecase
	outbox  OutboxFlusher
	payment PaymentFixer
}

func NewCompositeExecutor(order orderdomain.OrderCommandUsecase, outbox OutboxFlusher, payment PaymentFixer) *CompositeExecutor {
	return &CompositeExecutor{order: order, outbox: outbox, payment: payment}
}

func (e *CompositeExecutor) Execute(ctx context.Context, item domain.TaskItem) error {
	if item.TaskID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty taskId", nil)
	}
	switch item.JobType {
	case domain.JobOrderTimeoutCancel:
		return e.execOrderTimeoutCancel(ctx, item)
	case domain.JobPaymentStateFix:
		return e.execPaymentStateFix(ctx, item)
	case domain.JobOutboxRetry:
		return e.execOutboxRetry(ctx)
	default:
		return domain.NewBizError(domain.CodeInvalidArgument, "unsupported job type", nil)
	}
}

func (e *CompositeExecutor) execOrderTimeoutCancel(ctx context.Context, item domain.TaskItem) error {
	if e.order == nil {
		return domain.NewBizError(domain.CodeInternal, "order usecase not initialized", nil)
	}
	orderID, err := parseOrderID(item)
	if err != nil {
		return err
	}
	_, err = e.order.CancelOrder(ctx, orderdomain.CancelOrderCommand{
		OrderID:        orderID,
		IdempotencyKey: "compensation:order_timeout_cancel:" + item.TaskID,
	})
	return err
}

func (e *CompositeExecutor) execPaymentStateFix(ctx context.Context, item domain.TaskItem) error {
	if e.payment != nil {
		return e.payment.RepairPaymentState(ctx, item)
	}
	if e.order == nil {
		return domain.NewBizError(domain.CodeInternal, "payment fix usecase not initialized", nil)
	}
	orderID, err := parseOrderID(item)
	if err != nil {
		return err
	}
	_, err = e.order.TransitStatus(ctx, orderdomain.TransitStatusCommand{
		OrderID:        orderID,
		From:           orderdomain.OrderStatusPendingPay,
		To:             orderdomain.OrderStatusPaid,
		IdempotencyKey: "compensation:payment_fix:" + item.TaskID,
	})
	return err
}

func (e *CompositeExecutor) execOutboxRetry(ctx context.Context) error {
	if e.outbox == nil {
		return domain.NewBizError(domain.CodeInternal, "outbox usecase not initialized", nil)
	}
	return e.outbox.FlushOutbox(ctx)
}

func parseOrderID(item domain.TaskItem) (int64, error) {
	if id := parseOrderIDFromBizKey(item.BizKey); id > 0 {
		return id, nil
	}
	if len(item.Payload) == 0 {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "missing order id payload", nil)
	}
	var payload map[string]any
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid payload", err)
	}
	if raw, ok := payload["orderId"]; ok {
		switch v := raw.(type) {
		case float64:
			if v > 0 {
				return int64(v), nil
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	if raw, ok := payload["order_id"]; ok {
		switch v := raw.(type) {
		case float64:
			if v > 0 {
				return int64(v), nil
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid order id", nil)
}

func parseOrderIDFromBizKey(bizKey string) int64 {
	if strings.TrimSpace(bizKey) == "" {
		return 0
	}
	parts := strings.Split(strings.TrimSpace(bizKey), ":")
	if len(parts) == 0 {
		return 0
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func BuildTaskPayloadForOrder(orderID int64) []byte {
	if orderID <= 0 {
		return nil
	}
	out, _ := json.Marshal(map[string]any{
		"orderId": orderID,
		"ts":      time.Now().UnixMilli(),
	})
	return out
}
