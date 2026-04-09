package app

import (
	"context"
	"encoding/json"

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
		cached, found, getErr := s.deps.Idempotency.GetDoneResult(ctx, scene, key)
		if getErr != nil {
			return nil, getErr
		}
		if found && len(cached) > 0 {
			var out domain.OrderView
			if unmarshalErr := json.Unmarshal(cached, &out); unmarshalErr == nil {
				return &out, nil
			}
		}
		return nil, domain.NewBizError(domain.CodeConflict, "duplicated request", nil)
	}
	out, runErr := action(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, scene, key, token, runErr.Error())
		return nil, runErr
	}
	payload := []byte(nil)
	if out != nil {
		if b, marshalErr := json.Marshal(out); marshalErr == nil {
			payload = b
		}
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, scene, key, token, payload); doneErr != nil {
		_ = doneErr
	}
	return out, nil
}
