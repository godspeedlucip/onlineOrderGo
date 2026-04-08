package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	identityapp "go-baseline-skeleton/internal/identity/app"
	"go-baseline-skeleton/internal/identity/domain"
	"go-baseline-skeleton/internal/identity/infra/contextx"
	"go-baseline-skeleton/internal/identity/infra/jwt"
	"go-baseline-skeleton/internal/identity/infra/middleware"
	"go-baseline-skeleton/internal/identity/infra/password"
	"go-baseline-skeleton/internal/identity/infra/repo"
	"go-baseline-skeleton/internal/identity/infra/session"
	identitytx "go-baseline-skeleton/internal/identity/infra/tx"
)

type envelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

type loginData struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

func TestIdentityContract_LoginMeLogoutRevoke(t *testing.T) {
	h := setupIdentityHandler(t, 10*time.Minute, true)

	token := mustLogin(t, h, "admin", "123456")
	if status := callMe(t, h, token); status != http.StatusOK {
		t.Fatalf("me status mismatch before logout: got=%d want=%d", status, http.StatusOK)
	}
	if status := callLogout(t, h, token); status != http.StatusOK {
		t.Fatalf("logout status mismatch: got=%d want=%d", status, http.StatusOK)
	}
	if status := callMe(t, h, token); status != http.StatusUnauthorized {
		t.Fatalf("token should be revoked after logout: got=%d want=%d", status, http.StatusUnauthorized)
	}
}

func TestIdentityContract_TokenExpired(t *testing.T) {
	h := setupIdentityHandler(t, 50*time.Millisecond, false)
	token := mustLogin(t, h, "admin", "123456")
	time.Sleep(80 * time.Millisecond)

	if status := callMe(t, h, token); status != http.StatusUnauthorized {
		t.Fatalf("expired token should be unauthorized: got=%d want=%d", status, http.StatusUnauthorized)
	}
}

func TestIdentityContract_RevokeAllOnLogout(t *testing.T) {
	h := setupIdentityHandler(t, 10*time.Minute, true)

	token1 := mustLogin(t, h, "admin", "123456")
	token2 := mustLogin(t, h, "admin", "123456")

	if status := callLogout(t, h, token1); status != http.StatusOK {
		t.Fatalf("logout status mismatch: got=%d want=%d", status, http.StatusOK)
	}
	if status := callMe(t, h, token2); status != http.StatusUnauthorized {
		t.Fatalf("token2 should be invalid after revoke-all logout: got=%d want=%d", status, http.StatusUnauthorized)
	}
}

func setupIdentityHandler(t *testing.T, employeeTTL time.Duration, revokeAll bool) http.Handler {
	t.Helper()

	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	t.Cleanup(mini.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	principalCtx := contextx.NewPrincipalStore()
	authSvc := identityapp.NewAuthService(identityapp.AuthDeps{
		Repo: repo.NewInMemoryAccountRepo([]*domain.Account{
			{
				ID:           1,
				Type:         domain.AccountTypeEmployee,
				Username:     "admin",
				DisplayName:  "Admin",
				PasswordHash: password.HashMD5("123456"),
				Status:       domain.AccountStatusEnabled,
			},
		}),
		Token: jwt.NewTokenService(jwt.Config{
			Algorithm: "HS256",
			Employee: jwt.AccountJWTConfig{Secret: "itcast", TTL: employeeTTL, ClaimKey: "empId"},
			User:     jwt.AccountJWTConfig{Secret: "itcast", TTL: 10 * time.Minute, ClaimKey: "userId"},
		}),
		Password:          password.NewMD5Comparator(),
		PrincipalCtx:      principalCtx,
		Tx:                identitytx.NewNoopManager(),
		Sessions:          session.NewRedisStore(redisClient, "identity:test"),
		RevokeAllOnLogout: revokeAll,
	})

	authMiddleware := middleware.NewRequireAuth(authSvc, principalCtx)
	return NewHandler(authSvc, principalCtx, authMiddleware).Routes()
}

func mustLogin(t *testing.T, h http.Handler, identifier, pwd string) string {
	t.Helper()
	body := map[string]any{
		"accountType": "employee",
		"identifier":  identifier,
		"password":    pwd,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/identity/login", bytes.NewReader(b))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	var data loginData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode login data failed: %v", err)
	}
	if data.Token == "" {
		t.Fatal("empty token")
	}
	return data.Token
}

func callMe(t *testing.T, h http.Handler, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/identity/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func callLogout(t *testing.T, h http.Handler, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/identity/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}
