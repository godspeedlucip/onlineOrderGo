package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type Handler struct {
	usecase domain.PaymentCallbackUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(usecase domain.PaymentCallbackUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/payment/callback", http.HandlerFunc(h.callback))
	return mux
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "read body failed", err), http.StatusBadRequest)
		return
	}
	in := domain.CallbackInput{
		Headers:  flattenHeaders(r.Header),
		Body:     body,
		RemoteIP: r.RemoteAddr,
	}

	ack, bizErr := h.usecase.HandleCallback(r.Context(), in)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}

	if ack == nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInternal, "empty callback ack", nil), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(ack.HTTPStatus)
	_, _ = w.Write([]byte(ack.Body))
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func (h *Handler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			h.writeError(ctx, w, err, http.StatusBadRequest)
		case domain.CodeUnauthorized:
			h.writeError(ctx, w, err, http.StatusUnauthorized)
		case domain.CodeConflict:
			h.writeError(ctx, w, err, http.StatusConflict)
		default:
			h.writeError(ctx, w, err, http.StatusInternalServerError)
		}
		return
	}
	h.writeError(ctx, w, err, http.StatusInternalServerError)
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