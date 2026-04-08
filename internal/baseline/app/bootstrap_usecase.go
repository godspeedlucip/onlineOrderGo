package app

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go-baseline-skeleton/internal/baseline/domain"
)

type BootstrapUsecase struct {
	tx          domain.TxManager
	logger      domain.Logger
	cfg         *domain.Config
	repo        domain.RepositoryPort
	cache       domain.CachePort
	mq          domain.MQPort
	websocket   domain.WebSocketPort
	payment     domain.PaymentPort
	idempotency domain.IdempotencyStore
}

func NewBootstrapUsecase(
	tx domain.TxManager,
	logger domain.Logger,
	cfg *domain.Config,
	repo domain.RepositoryPort,
	cache domain.CachePort,
	mq domain.MQPort,
	websocket domain.WebSocketPort,
	payment domain.PaymentPort,
	idempotency domain.IdempotencyStore,
) *BootstrapUsecase {
	return &BootstrapUsecase{
		tx:          tx,
		logger:      logger,
		cfg:         cfg,
		repo:        repo,
		cache:       cache,
		mq:          mq,
		websocket:   websocket,
		payment:     payment,
		idempotency: idempotency,
	}
}

type HealthOutput struct {
	Status string `json:"status"`
	Env    string `json:"env"`
}

func (u *BootstrapUsecase) ValidateStartup(ctx context.Context) error {
	if u.cfg == nil {
		return domain.NewBizError(domain.CodeInternal, "config not initialized", nil)
	}
	if u.cfg.App.Name == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "app.name is required", nil)
	}
	if u.cfg.HTTP.Addr == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "http.addr is required", nil)
	}
	if u.cfg.Idempotency.TTLSecond <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "idempotency.ttl_second must be > 0", nil)
	}
	return nil
}

func (u *BootstrapUsecase) Health(ctx context.Context) (*HealthOutput, error) {
	if err := u.ValidateStartup(ctx); err != nil {
		return nil, err
	}

	if u.repo != nil {
		if err := u.repo.Ping(ctx); err != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "repository unavailable", err)
		}
	}
	if u.cache != nil {
		if err := u.cache.Ping(ctx); err != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "cache unavailable", err)
		}
	}
	if u.mq != nil {
		if err := u.mq.Ping(ctx); err != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "mq unavailable", err)
		}
	}
	if u.websocket != nil {
		if err := u.websocket.Ping(ctx); err != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "websocket unavailable", err)
		}
	}
	if u.payment != nil {
		if err := u.payment.Ping(ctx); err != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "payment unavailable", err)
		}
	}

	return &HealthOutput{Status: "ok", Env: u.cfg.App.Env}, nil
}

// ExampleTxBoundary shows transaction control in app layer.
func (u *BootstrapUsecase) ExampleTxBoundary(ctx context.Context) error {
	if u.tx == nil {
		return errors.New("tx manager is nil")
	}

	return u.tx.RunInTx(ctx, func(txCtx context.Context) error {
		u.logger.Info(txCtx, "tx_boundary_enter", nil)
		// TODO: place domain operations that must share one transaction.
		return nil
	})
}

type IdempotentOutput struct {
	Result         map[string]any `json:"result"`
	FromIdempotent bool           `json:"fromIdempotent"`
}

// ExecuteIdempotent wraps a business action with idempotency + tx boundary.
// It is intentionally generic so order/create can reuse this path later.
func (u *BootstrapUsecase) ExecuteIdempotent(
	ctx context.Context,
	scene string,
	key string,
	action func(ctx context.Context) (map[string]any, error),
) (*IdempotentOutput, error) {
	if scene == "" || key == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "scene and key are required", nil)
	}
	if action == nil {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "action is required", nil)
	}
	if u.tx == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "tx manager is nil", nil)
	}
	if !u.cfg.Idempotency.Enabled || u.idempotency == nil {
		// TODO: confirm with Java: whether idempotency-disabled requests should be blocked.
		result, err := u.runActionInTx(ctx, action)
		if err != nil {
			return nil, err
		}
		return &IdempotentOutput{Result: result, FromIdempotent: false}, nil
	}

	ttl := time.Duration(u.cfg.Idempotency.TTLSecond) * time.Second
	token, acquired, err := u.idempotency.Acquire(ctx, scene, key, ttl)
	if err != nil {
		return nil, domain.NewBizError(domain.CodeServiceUnavailable, "idempotency store acquire failed", err)
	}

	if !acquired {
		rec, getErr := u.idempotency.Get(ctx, scene, key)
		if getErr != nil {
			return nil, domain.NewBizError(domain.CodeServiceUnavailable, "idempotency store read failed", getErr)
		}
		if rec == nil {
			return nil, domain.NewBizError(domain.CodeConflict, "duplicate request in progress", nil)
		}
		switch rec.Status {
		case domain.StatusSucceeded:
			cached := map[string]any{}
			if len(rec.Payload) > 0 {
				if unmarshalErr := json.Unmarshal(rec.Payload, &cached); unmarshalErr != nil {
					return nil, domain.NewBizError(domain.CodeInternal, "idempotency payload decode failed", unmarshalErr)
				}
			}
			return &IdempotentOutput{Result: cached, FromIdempotent: true}, nil
		case domain.StatusProcessing:
			return nil, domain.NewBizError(domain.CodeConflict, "duplicate request in progress", nil)
		default:
			// FAILED or unexpected states are retried by client with a new request cycle.
			return nil, domain.NewBizError(domain.CodeConflict, "request needs retry", nil)
		}
	}

	result, bizErr := u.runActionInTx(ctx, action)
	if bizErr != nil {
		_, markErr := u.idempotency.MarkFailed(ctx, scene, key, token, bizErr.Error())
		if markErr != nil {
			u.logger.Error(ctx, "idempotency_mark_failed", markErr, map[string]any{"scene": scene})
		}
		return nil, bizErr
	}

	payload, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		_, markErr := u.idempotency.MarkFailed(ctx, scene, key, token, marshalErr.Error())
		if markErr != nil {
			u.logger.Error(ctx, "idempotency_mark_failed", markErr, map[string]any{"scene": scene})
		}
		return nil, domain.NewBizError(domain.CodeInternal, "idempotency payload encode failed", marshalErr)
	}

	updated, markSuccessErr := u.idempotency.MarkSuccess(ctx, scene, key, token, payload)
	if markSuccessErr != nil {
		return nil, domain.NewBizError(domain.CodeServiceUnavailable, "idempotency store update failed", markSuccessErr)
	}
	if !updated {
		return nil, domain.NewBizError(domain.CodeConflict, "idempotency state changed by another request", nil)
	}

	return &IdempotentOutput{Result: result, FromIdempotent: false}, nil
}

func (u *BootstrapUsecase) runActionInTx(ctx context.Context, action func(ctx context.Context) (map[string]any, error)) (map[string]any, error) {
	result := map[string]any{}
	err := u.tx.RunInTx(ctx, func(txCtx context.Context) error {
		out, actionErr := action(txCtx)
		if actionErr != nil {
			return actionErr
		}
		result = out
		return nil
	})
	if err != nil {
		var bizErr *domain.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, domain.NewBizError(domain.CodeInternal, "transaction execution failed", err)
	}
	return result, nil
}
