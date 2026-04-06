package app

import (
	"context"
	"strings"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type Deps struct {
	Verifier    domain.PaymentVerifier
	Repo        domain.PaymentRepository
	Idempotency domain.CallbackIdempotencyStore
	Publisher   domain.PaymentEventPublisher
	GrayPolicy  domain.GrayPolicy
	Tx          domain.TxManager

	// Optional cross-domain dependencies. Keep injected for future expansion.
	Cache     domain.CachePort
	MQ        domain.MQPort
	WebSocket domain.WebSocketPort
	Payment   domain.PaymentPort

	IdempotencyTTL time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.IdempotencyTTL <= 0 {
		deps.IdempotencyTTL = 10 * time.Minute
	}
	return &Service{deps: deps}
}

func (s *Service) HandleCallback(ctx context.Context, in domain.CallbackInput) (*domain.CallbackAck, error) {
	if s.deps.Verifier == nil || s.deps.Repo == nil || s.deps.Tx == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "callback dependencies not initialized", nil)
	}
	if len(in.Body) == 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty callback body", nil)
	}

	verified, err := s.deps.Verifier.VerifyAndParse(ctx, in.Headers, in.Body)
	if err != nil {
		_ = s.deps.Repo.InsertCallbackLog(ctx, domain.CallbackLog{
			NotifyID:     "",
			Channel:      "",
			HTTPHeaders:  in.Headers,
			Body:         in.Body,
			Verified:     false,
			ErrorMessage: err.Error(),
			CreatedAt:    time.Now(),
		})
		return nil, domain.NewBizError(domain.CodeUnauthorized, "callback signature verify failed", err)
	}
	if vErr := validateVerifiedCallback(verified); vErr != nil {
		return nil, vErr
	}

	if s.deps.GrayPolicy != nil {
		decision, dErr := s.deps.GrayPolicy.Decide(ctx, verified)
		if dErr != nil {
			return nil, dErr
		}
		if !decision.Enabled {
			// Gray disabled: ack success to avoid provider retry storm.
			// TODO: align with Java fallback strategy for bypass traffic.
			return &domain.CallbackAck{HTTPStatus: 200, Body: "success"}, nil
		}
	}
	if !isPaidSuccessStatus(verified.RawStatus) {
		// Keep callback ACK success for non-paid status to avoid provider retries.
		// TODO: align exact status-mapping with Java channel enum definitions.
		_ = s.deps.Repo.InsertCallbackLog(ctx, domain.CallbackLog{
			NotifyID:      verified.NotifyID,
			OrderNo:       verified.OrderNo,
			TransactionNo: verified.TransactionNo,
			Channel:       verified.Channel,
			HTTPHeaders:   in.Headers,
			Body:          in.Body,
			Verified:      true,
			ErrorMessage:  "ignored by status: " + verified.RawStatus,
			CreatedAt:     time.Now(),
		})
		return &domain.CallbackAck{HTTPStatus: 200, Body: "success"}, nil
	}

	return s.withIdempotency(ctx, "payment_callback", buildIdempotencyKey(verified), func(txCtx context.Context) (*domain.CallbackAck, error) {
		order, qErr := s.deps.Repo.GetOrderByNo(txCtx, verified.OrderNo)
		if qErr != nil {
			return nil, qErr
		}
		if order == nil {
			return nil, domain.NewBizError(domain.CodeInvalidArgument, "order not found", nil)
		}
		if order.TotalAmount != verified.PaidAmount {
			return nil, domain.NewBizError(domain.CodeConflict, "paid amount mismatch", nil)
		}
		if order.MerchantID != "" && verified.MerchantID != "" && order.MerchantID != verified.MerchantID {
			return nil, domain.NewBizError(domain.CodeConflict, "merchant mismatch", nil)
		}

		updated, upErr := s.deps.Repo.UpdateOrderPaidIfPending(txCtx, order.OrderID, verified.PaidAt, verified.TransactionNo, verified.PaidAmount)
		if upErr != nil {
			return nil, upErr
		}
		if updated {
			if err := s.deps.Repo.InsertPaymentRecord(txCtx, domain.PaymentRecord{
				OrderID:       order.OrderID,
				OrderNo:       order.OrderNo,
				TransactionNo: verified.TransactionNo,
				Channel:       verified.Channel,
				PaidAmount:    verified.PaidAmount,
				PaidAt:        verified.PaidAt,
				RawStatus:     verified.RawStatus,
			}); err != nil {
				return nil, err
			}
		}

		if err := s.deps.Repo.InsertCallbackLog(txCtx, domain.CallbackLog{
			NotifyID:      verified.NotifyID,
			OrderNo:       verified.OrderNo,
			TransactionNo: verified.TransactionNo,
			Channel:       verified.Channel,
			HTTPHeaders:   in.Headers,
			Body:          in.Body,
			Verified:      true,
			ErrorMessage:  "",
			CreatedAt:     time.Now(),
		}); err != nil {
			return nil, err
		}

		if updated && s.deps.Publisher != nil {
			_ = s.deps.Publisher.PublishOrderPaid(txCtx, domain.OrderPaidEvent{
				OrderID:       order.OrderID,
				OrderNo:       order.OrderNo,
				TransactionNo: verified.TransactionNo,
				PaidAmount:    verified.PaidAmount,
				PaidAt:        verified.PaidAt,
			})
			// TODO: decide retry/outbox strategy if publish fails.
		}

		return &domain.CallbackAck{HTTPStatus: 200, Body: "success"}, nil
	})
}

func validateVerifiedCallback(cb *domain.VerifiedCallback) error {
	if cb == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty verified callback", nil)
	}
	if cb.OrderNo == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty orderNo", nil)
	}
	if cb.TransactionNo == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty transactionNo", nil)
	}
	if cb.PaidAmount <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid paidAmount", nil)
	}
	return nil
}

func buildIdempotencyKey(cb *domain.VerifiedCallback) string {
	// Prefer notify id; fallback to channel + transaction + order for channels that don't provide notify id.
	if cb == nil {
		return ""
	}
	if cb.NotifyID != "" {
		return cb.NotifyID
	}
	return cb.Channel + "|" + cb.TransactionNo + "|" + cb.OrderNo
}

func isPaidSuccessStatus(raw string) bool {
	s := strings.ToUpper(strings.TrimSpace(raw))
	return s == "SUCCESS" || s == "TRADE_SUCCESS" || s == "PAID"
}

func (s *Service) withIdempotency(
	ctx context.Context,
	scene string,
	idemKey string,
	action func(ctx context.Context) (*domain.CallbackAck, error),
) (*domain.CallbackAck, error) {
	if action == nil {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "action is nil", nil)
	}

	runInTx := func(runCtx context.Context) (*domain.CallbackAck, error) {
		var out *domain.CallbackAck
		err := s.deps.Tx.RunInTx(runCtx, func(txCtx context.Context) error {
			ret, e := action(txCtx)
			if e != nil {
				return e
			}
			out = ret
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	if s.deps.Idempotency == nil || idemKey == "" {
		return runInTx(ctx)
	}

	token, acquired, err := s.deps.Idempotency.Acquire(ctx, scene, idemKey, s.deps.IdempotencyTTL)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return &domain.CallbackAck{HTTPStatus: 200, Body: "success"}, nil
	}

	out, runErr := runInTx(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, scene, idemKey, token, runErr.Error())
		return nil, runErr
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, scene, idemKey, token); doneErr != nil {
		// TODO: add retry path for mark-done failure.
		_ = doneErr
	}
	return out, nil
}
