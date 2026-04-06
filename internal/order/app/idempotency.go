package app

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

func (s *Service) withIdempotency(
	ctx context.Context,
	scene string,
	key string,
	action func(ctx context.Context) (*domain.OrderView, error),
) (*domain.OrderView, error) {
	if action == nil {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "action is nil", nil)
	}
	if s.deps.Idempotency == nil || key == "" {
		return action(ctx)
	}
	token, acquired, err := s.deps.Idempotency.Acquire(ctx, scene, key, s.deps.IdempotencyTTL)
	if err != nil {
		return nil, err
	}
	if !acquired {
		// TODO: optionally return latest order snapshot instead of conflict.
		return nil, domain.NewBizError(domain.CodeConflict, "duplicated request", nil)
	}
	out, runErr := action(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, scene, key, token, runErr.Error())
		return nil, runErr
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, scene, key, token); doneErr != nil {
		// TODO: add async retry for mark-done failure.
		_ = doneErr
	}
	return out, nil
}
