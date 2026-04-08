package jwt

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

func TestTokenService_IssueAndParse_WithJavaClaimKeys(t *testing.T) {
	svc := NewTokenService(Config{
		Algorithm: "HS256",
		Employee: AccountJWTConfig{Secret: "itcast", Issuer: "admin-issuer", TTL: time.Minute, ClaimKey: "empId"},
		User:     AccountJWTConfig{Secret: "itcast", Issuer: "user-issuer", TTL: time.Minute, ClaimKey: "userId"},
	})

	token, _, err := svc.Issue(context.Background(), domain.Claims{
		AccountID:    101,
		AccountType:  domain.AccountTypeEmployee,
		TokenID:      "tid-1",
		TokenVersion: 1,
		IssuedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if !strings.Contains(token, ".") {
		t.Fatalf("invalid token format: %s", token)
	}

	claims, err := svc.Parse(context.Background(), token)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if claims.AccountType != domain.AccountTypeEmployee || claims.AccountID != 101 {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestTokenService_Parse_InvalidIssuer(t *testing.T) {
	issuerA := NewTokenService(Config{
		Algorithm: "HS256",
		Employee:  AccountJWTConfig{Secret: "itcast", Issuer: "issuer-A", TTL: time.Minute, ClaimKey: "empId"},
		User:      AccountJWTConfig{Secret: "itcast", Issuer: "issuer-A", TTL: time.Minute, ClaimKey: "userId"},
	})
	issuerB := NewTokenService(Config{
		Algorithm: "HS256",
		Employee:  AccountJWTConfig{Secret: "itcast", Issuer: "issuer-B", TTL: time.Minute, ClaimKey: "empId"},
		User:      AccountJWTConfig{Secret: "itcast", Issuer: "issuer-B", TTL: time.Minute, ClaimKey: "userId"},
	})

	token, _, err := issuerA.Issue(context.Background(), domain.Claims{AccountID: 1, AccountType: domain.AccountTypeEmployee, IssuedAt: time.Now()})
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if _, err := issuerB.Parse(context.Background(), token); err == nil {
		t.Fatal("expected invalid issuer error")
	}
}
