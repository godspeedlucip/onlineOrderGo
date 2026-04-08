package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type RequireAuth struct {
	auth         domain.AuthUsecase
	principalCtx domain.PrincipalContext
}

type errResp struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func NewRequireAuth(auth domain.AuthUsecase, principalCtx domain.PrincipalContext) *RequireAuth {
	return &RequireAuth{auth: auth, principalCtx: principalCtx}
}

func (m *RequireAuth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeUnauthorized(w, "missing token")
			return
		}

		principal, err := m.auth.VerifyToken(r.Context(), token)
		if err != nil {
			writeUnauthorized(w, "invalid token")
			return
		}

		ctx := m.principalCtx.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractToken(r *http.Request) string {
	if token := BearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.Header.Get("token")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.Header.Get("authentication")); token != "" {
		return token
	}
	return ""
}

func BearerToken(v string) string {
	parts := strings.SplitN(strings.TrimSpace(v), " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(errResp{
		Code:      string(domain.CodeUnauthorized),
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	})
}
