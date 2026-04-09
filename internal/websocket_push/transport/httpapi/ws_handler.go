package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Handler struct {
	sessionUsecase domain.SessionUsecase
	pushUsecase    domain.PushUsecase
	authPort       domain.AuthPort
	wsServer       http.Handler
	offlinePuller  interface {
		PullOffline(ctx context.Context, userID int64, limit int) ([]domain.PushMessage, error)
	}
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(sessionUsecase domain.SessionUsecase, pushUsecase domain.PushUsecase, authPort domain.AuthPort) *Handler {
	return &Handler{sessionUsecase: sessionUsecase, pushUsecase: pushUsecase, authPort: authPort}
}

func (h *Handler) SetWebSocketServer(server http.Handler) {
	h.wsServer = server
}

func (h *Handler) SetOfflinePuller(puller interface {
	PullOffline(ctx context.Context, userID int64, limit int) ([]domain.PushMessage, error)
}) {
	h.offlinePuller = puller
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/ws/connect", http.HandlerFunc(h.connect))
	mux.Handle("/ws/disconnect", http.HandlerFunc(h.disconnect))
	mux.Handle("/ws/heartbeat", http.HandlerFunc(h.heartbeat))
	mux.Handle("/ws/push", http.HandlerFunc(h.push))
	mux.Handle("/ws/offline/pull", http.HandlerFunc(h.pullOffline))
	if h.wsServer != nil {
		mux.Handle("/ws", h.wsServer)
	}
	return mux
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token      string            `json:"token"`
		ClientType domain.ClientType `json:"clientType"`
		UserID     int64             `json:"userId"`
		ShopID     int64             `json:"shopId"`
		Channels   []string          `json:"channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	out, err := h.sessionUsecase.Connect(r.Context(), domain.ConnectRequest{
		Token:      req.Token,
		ClientType: req.ClientType,
		UserID:     req.UserID,
		ShopID:     req.ShopID,
		Channels:   req.Channels,
		RemoteIP:   r.RemoteAddr,
	})
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	if err := h.sessionUsecase.Disconnect(r.Context(), req.SessionID); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	if err := h.sessionUsecase.Heartbeat(r.Context(), req.SessionID); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) push(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		MessageID  string            `json:"messageId"`
		EventType  string            `json:"eventType"`
		TargetUser int64             `json:"targetUser"`
		TargetShop int64             `json:"targetShop"`
		Channel    string            `json:"channel"`
		PayloadB64 string            `json:"payloadBase64"`
		Metadata   map[string]string `json:"metadata"`
		Broadcast  bool              `json:"broadcast"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	payload, err := base64.StdEncoding.DecodeString(req.PayloadB64)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid payloadBase64", err), http.StatusBadRequest)
		return
	}
	msg := domain.PushMessage{
		MessageID:  req.MessageID,
		EventType:  req.EventType,
		TargetUser: req.TargetUser,
		TargetShop: req.TargetShop,
		Channel:    req.Channel,
		Payload:    payload,
		Metadata:   req.Metadata,
		CreatedAt:  time.Now(),
	}
	var out *domain.PushResult
	if req.Broadcast {
		out, err = h.pushUsecase.Broadcast(r.Context(), msg)
	} else {
		out, err = h.pushUsecase.Push(r.Context(), msg)
	}
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: out, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) pullOffline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token string `json:"token"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}
	if h.authPort == nil || h.offlinePuller == nil {
		h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: []domain.PushMessage{}, Timestamp: time.Now().UnixMilli()})
		return
	}
	uid, err := h.authPort.ValidateToken(r.Context(), strings.TrimSpace(req.Token))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	messages, err := h.offlinePuller.PullOffline(r.Context(), uid, req.Limit)
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, response{Code: "SUCCESS", Message: "ok", Data: messages, Timestamp: time.Now().UnixMilli()})
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
