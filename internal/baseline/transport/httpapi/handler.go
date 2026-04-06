package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/baseline/app"
	"go-baseline-skeleton/internal/baseline/domain"
	infraerrors "go-baseline-skeleton/internal/baseline/infra/errors"
	"go-baseline-skeleton/internal/baseline/infra/logging"
)

type Handler struct {
	usecase *app.BootstrapUsecase
	logger  domain.Logger
	mapper  *infraerrors.HTTPErrorMapper
}

type Response struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Data      any         `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	TraceID   string      `json:"traceId,omitempty"`
}

func NewHandler(usecase *app.BootstrapUsecase, logger domain.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		logger:  logger,
		mapper:  infraerrors.NewHTTPErrorMapper(),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", h.wrap(h.health))
	mux.Handle("/baseline/idempotency/demo", h.wrap(h.idempotencyDemo))
	return mux
}

func (h *Handler) wrap(next func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ctx := logging.ContextWithRequestID(r.Context(), requestIDFromHeader(r))
		r = r.WithContext(ctx)

		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("panic recovered: %v", rec)
				h.writeError(ctx, w, domain.NewBizError(domain.CodeInternal, "internal panic", err))
			}
			h.logger.Info(ctx, "http_request", map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"latency_ms":  time.Since(started).Milliseconds(),
				"remote_addr": r.RemoteAddr,
			})
		}()

		if err := next(w, r); err != nil {
			h.writeError(ctx, w, err)
		}
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out, err := h.usecase.Health(ctx)
	if err != nil {
		return err
	}

	h.writeJSON(ctx, w, http.StatusOK, Response{
		Code:      "0",
		Message:   "success",
		Data:      out,
		Timestamp: time.Now().UnixMilli(),
		TraceID:   requestIDFromContext(ctx),
	})
	return nil
}

func (h *Handler) idempotencyDemo(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil)
	}

	ctx := r.Context()
	idKey := r.Header.Get("Idempotency-Key")
	if idKey == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "missing Idempotency-Key", nil)
	}

	out, err := h.usecase.ExecuteIdempotent(ctx, "baseline_demo", idKey, func(txCtx context.Context) (map[string]any, error) {
		// TODO: replace this demo action with real business action from order/create chain.
		return map[string]any{
			"acceptedAt": time.Now().Format(time.RFC3339Nano),
			"txScope":    "app-layer",
			"note":       "baseline idempotent action executed",
		}, nil
	})
	if err != nil {
		return err
	}

	h.writeJSON(ctx, w, http.StatusOK, Response{
		Code:      "0",
		Message:   "success",
		Data:      out,
		Timestamp: time.Now().UnixMilli(),
		TraceID:   requestIDFromContext(ctx),
	})
	return nil
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error) {
	status, code, message := h.mapper.ToHTTP(err)
	h.logger.Error(ctx, "request_failed", err, map[string]any{"code": code, "status": status})

	h.writeJSON(ctx, w, status, Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
		TraceID:   requestIDFromContext(ctx),
	})
}

func (h *Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error(ctx, "encode_response_failed", err, nil)
	}
}

func requestIDFromHeader(r *http.Request) string {
	id := r.Header.Get("X-Request-Id")
	if id == "" {
		id = r.Header.Get("X-Trace-Id")
	}
	if id == "" {
		id = fmt.Sprintf("generated-%d", time.Now().UnixNano())
	}
	return id
}

func requestIDFromContext(ctx context.Context) string {
	id, ok := logging.RequestIDFromContext(ctx)
	if !ok {
		return ""
	}
	return id
}