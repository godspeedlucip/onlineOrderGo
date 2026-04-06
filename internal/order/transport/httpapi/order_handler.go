package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

type Handler struct {
	usecase domain.OrderCommandUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(usecase domain.OrderCommandUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/orders", http.HandlerFunc(h.createOrder))
	mux.Handle("/orders/cancel", http.HandlerFunc(h.cancelOrder))
	mux.Handle("/orders/transit", http.HandlerFunc(h.transitStatus))
	return mux
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID         int64  `json:"userId"`
		AddressID      int64  `json:"addressId"`
		Remark         string `json:"remark"`
		PaymentMethod  string `json:"paymentMethod"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	out, err := h.usecase.CreateOrder(r.Context(), domain.CreateOrderCommand{
		UserID:         req.UserID,
		AddressID:      req.AddressID,
		Remark:         req.Remark,
		PaymentMethod:  req.PaymentMethod,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		OrderID        int64  `json:"orderId"`
		OperatorID     int64  `json:"operatorId"`
		Reason         string `json:"reason"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	out, err := h.usecase.CancelOrder(r.Context(), domain.CancelOrderCommand{
		OrderID:        req.OrderID,
		OperatorID:     req.OperatorID,
		Reason:         req.Reason,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) transitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		OrderID        int64              `json:"orderId"`
		From           domain.OrderStatus `json:"from"`
		To             domain.OrderStatus `json:"to"`
		TriggerSource  string             `json:"triggerSource"`
		IdempotencyKey string             `json:"idempotencyKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	out, err := h.usecase.TransitStatus(r.Context(), domain.TransitStatusCommand{
		OrderID:        req.OrderID,
		From:           req.From,
		To:             req.To,
		TriggerSource:  req.TriggerSource,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			h.writeError(ctx, w, err, http.StatusBadRequest)
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
