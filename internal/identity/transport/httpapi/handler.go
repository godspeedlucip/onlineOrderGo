package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type Handler struct {
	auth         domain.AuthUsecase
	principalCtx domain.PrincipalContext
	middleware   domain.AuthMiddleware
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type loginRequest struct {
	AccountType string `json:"accountType"`
	Identifier  string `json:"identifier"`
	Password    string `json:"password"`
}

func NewHandler(auth domain.AuthUsecase, principalCtx domain.PrincipalContext, middleware domain.AuthMiddleware) *Handler {
	return &Handler{auth: auth, principalCtx: principalCtx, middleware: middleware}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/identity/login", http.HandlerFunc(h.login))
	mux.Handle("/identity/me", h.middleware.RequireAuth(http.HandlerFunc(h.me)))
	mux.Handle("/identity/logout", h.middleware.RequireAuth(http.HandlerFunc(h.logout)))
	return mux
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusBadRequest)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid request body", err), http.StatusBadRequest)
		return
	}

	out, err := h.auth.Login(r.Context(), domain.LoginInput{
		AccountType: domain.AccountType(req.AccountType),
		Identifier:  req.Identifier,
		Password:    req.Password,
	})
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}

	h.writeOK(r.Context(), w, out)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalCtx.Principal(r.Context())
	if !ok {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeUnauthorized, "unauthorized", nil), http.StatusUnauthorized)
		return
	}
	h.writeOK(r.Context(), w, p)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "method not allowed", nil), http.StatusBadRequest)
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeUnauthorized, "missing token", nil), http.StatusUnauthorized)
		return
	}
	if err := h.auth.Logout(r.Context(), token); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"loggedOut": true})
}

func (h *Handler) writeOK(ctx context.Context, w http.ResponseWriter, data any) {
	h.writeJSON(ctx, w, http.StatusOK, response{Code: "0", Message: "success", Data: data, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeUnauthorized:
			h.writeError(ctx, w, err, http.StatusUnauthorized)
		case domain.CodeForbidden:
			h.writeError(ctx, w, err, http.StatusForbidden)
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
	msg := "internal error"
	code := string(domain.CodeInternal)
	if bizErr, ok := err.(*domain.BizError); ok {
		msg = bizErr.Message
		code = string(bizErr.Code)
	}
	h.writeJSON(ctx, w, status, response{Code: code, Message: msg, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, resp response) {
	_ = ctx
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func bearerToken(v string) string {
	parts := strings.SplitN(strings.TrimSpace(v), " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}