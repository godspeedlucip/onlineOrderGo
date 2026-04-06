package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/cart/domain"
)

type Handler struct {
	cart domain.CartUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(cart domain.CartUsecase) *Handler {
	return &Handler{cart: cart}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/cart/add", http.HandlerFunc(h.add))
	mux.Handle("/cart/sub", http.HandlerFunc(h.sub))
	mux.Handle("/cart/update", http.HandlerFunc(h.update))
	mux.Handle("/cart/list", http.HandlerFunc(h.list))
	mux.Handle("/cart/clear", http.HandlerFunc(h.clear))
	return mux
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var req domain.AddCartItemCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	out, err := h.cart.AddItem(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) sub(w http.ResponseWriter, r *http.Request) {
	var req domain.SubCartItemCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	out, err := h.cart.SubItem(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateCartQtyCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	out, err := h.cart.UpdateQuantity(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.cart.ListItems(r.Context())
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	if err := h.cart.Clear(r.Context(), idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"cleared": true})
}

func idemKey(r *http.Request) string {
	return r.Header.Get("Idempotency-Key")
}

func (h *Handler) writeOK(ctx context.Context, w http.ResponseWriter, data any) {
	h.writeJSON(ctx, w, http.StatusOK, response{Code: "0", Message: "success", Data: data, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			h.writeError(ctx, w, err, http.StatusBadRequest)
		case domain.CodeNotFound:
			h.writeError(ctx, w, err, http.StatusNotFound)
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