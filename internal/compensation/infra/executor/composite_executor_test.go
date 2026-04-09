package executor

import (
	"context"
	"testing"

	"go-baseline-skeleton/internal/compensation/domain"
	orderdomain "go-baseline-skeleton/internal/order/domain"
)

type fakeOrderUsecase struct {
	cancelCalls  int
	transitCalls int
	lastCancel   orderdomain.CancelOrderCommand
	lastTransit  orderdomain.TransitStatusCommand
}

func (f *fakeOrderUsecase) CreateOrder(ctx context.Context, cmd orderdomain.CreateOrderCommand) (*orderdomain.OrderView, error) {
	_ = ctx
	_ = cmd
	return nil, nil
}

func (f *fakeOrderUsecase) CancelOrder(ctx context.Context, cmd orderdomain.CancelOrderCommand) (*orderdomain.OrderView, error) {
	_ = ctx
	f.cancelCalls++
	f.lastCancel = cmd
	return &orderdomain.OrderView{OrderID: cmd.OrderID, Status: orderdomain.OrderStatusCanceled}, nil
}

func (f *fakeOrderUsecase) TransitStatus(ctx context.Context, cmd orderdomain.TransitStatusCommand) (*orderdomain.OrderView, error) {
	_ = ctx
	f.transitCalls++
	f.lastTransit = cmd
	return &orderdomain.OrderView{OrderID: cmd.OrderID, Status: cmd.To}, nil
}

type fakeOutboxFlusher struct {
	calls int
}

func (f *fakeOutboxFlusher) FlushOutbox(ctx context.Context) error {
	_ = ctx
	f.calls++
	return nil
}

func TestCompositeExecutor_OrderTimeoutCancel_UseOrderUsecase(t *testing.T) {
	orderUC := &fakeOrderUsecase{}
	exec := NewCompositeExecutor(orderUC, nil, nil)
	err := exec.Execute(context.Background(), domain.TaskItem{
		TaskID:  "task_1",
		JobType: domain.JobOrderTimeoutCancel,
		BizKey:  "order:1001",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if orderUC.cancelCalls != 1 || orderUC.lastCancel.OrderID != 1001 {
		t.Fatalf("unexpected cancel call: %+v calls=%d", orderUC.lastCancel, orderUC.cancelCalls)
	}
}

func TestCompositeExecutor_PaymentStateFix_TransitToPaid(t *testing.T) {
	orderUC := &fakeOrderUsecase{}
	exec := NewCompositeExecutor(orderUC, nil, nil)
	err := exec.Execute(context.Background(), domain.TaskItem{
		TaskID:  "task_2",
		JobType: domain.JobPaymentStateFix,
		Payload: []byte(`{"orderId":2002}`),
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if orderUC.transitCalls != 1 {
		t.Fatalf("expected transit once, got=%d", orderUC.transitCalls)
	}
	if orderUC.lastTransit.OrderID != 2002 || orderUC.lastTransit.From != orderdomain.OrderStatusPendingPay || orderUC.lastTransit.To != orderdomain.OrderStatusPaid {
		t.Fatalf("unexpected transit cmd: %+v", orderUC.lastTransit)
	}
}

func TestCompositeExecutor_OutboxRetry_FlushOutbox(t *testing.T) {
	flusher := &fakeOutboxFlusher{}
	exec := NewCompositeExecutor(nil, flusher, nil)
	err := exec.Execute(context.Background(), domain.TaskItem{
		TaskID:  "task_3",
		JobType: domain.JobOutboxRetry,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if flusher.calls != 1 {
		t.Fatalf("expected one flush call, got=%d", flusher.calls)
	}
}
