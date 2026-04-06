package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-baseline-skeleton/internal/compensation/domain"
)

type Handler struct {
	usecase domain.CompensationUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(usecase domain.CompensationUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/compensation/run", http.HandlerFunc(h.runOnce))
	mux.Handle("/compensation/replay-failed", http.HandlerFunc(h.replayFailed))
	return mux
}

func (h *Handler) runOnce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	job := domain.JobType(r.URL.Query().Get("job"))
	out, err := h.usecase.RunOnce(r.Context(), job)
	if err != nil {
		h.writeError(r.Context(), w, err, http.StatusBadRequest)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) replayFailed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	out, err := h.usecase.ReplayFailed(r.Context(), limit)
	if err != nil {
		h.writeError(r.Context(), w, err, http.StatusBadRequest)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	_ = ctx
	code := string(domain.CodeInternal)
	msg := "internal error"
	if bizErr, ok := err.(*domain.BizError); ok {
		code = string(bizErr.Code)
		msg = bizErr.Message
	}
	h.writeJSON(ctx, w, status, response{Code: code, Message: msg, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, resp response) {
	_ = ctx
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
