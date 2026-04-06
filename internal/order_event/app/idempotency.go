package app

import (
	"context"

	"go-baseline-skeleton/internal/order_event/domain"
)

func (s *Service) withConsumeIdempotency(ctx context.Context, eventID string, action func(ctx context.Context) error) error {
	if action == nil {
		return domain.NewBizError(domain.CodeInvalidArgument, "action is nil", nil)
	}
	if s.deps.Idempotency == nil || eventID == "" {
		return action(ctx)
	}
	token, acquired, err := s.deps.Idempotency.Acquire(ctx, eventID, s.deps.ConsumeIdempotencyTTL)
	if err != nil {
		return err
	}
	if !acquired {
		// duplicated consumed message, safe to ack.
		return nil
	}
	runErr := action(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, eventID, token, runErr.Error())
		return runErr
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, eventID, token); doneErr != nil {
		// TODO: async retry mark done
		_ = doneErr
	}
	return nil
}
