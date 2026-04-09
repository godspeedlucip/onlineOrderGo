package ws

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenAuth_ValidateToken(t *testing.T) {
	auth := NewTokenAuth(JWTAuthConfig{
		Secret:    "secret",
		Algorithm: "HS256",
		Issuer:    "sky",
		Audience:  "ws",
		UserClaim: "userId",
	})
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": 123,
		"iss":    "sky",
		"aud":    "ws",
		"exp":    time.Now().Add(time.Minute).Unix(),
	})
	raw, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	uid, err := auth.ValidateToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if uid != 123 {
		t.Fatalf("expect userId 123, got %d", uid)
	}
}

func TestTokenAuth_ValidateTokenExpired(t *testing.T) {
	auth := NewTokenAuth(JWTAuthConfig{
		Secret:    "secret",
		Algorithm: "HS256",
		Issuer:    "sky",
		Audience:  "ws",
		UserClaim: "userId",
	})
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": 123,
		"iss":    "sky",
		"aud":    "ws",
		"exp":    time.Now().Add(-time.Minute).Unix(),
	})
	raw, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := auth.ValidateToken(context.Background(), raw); err == nil {
		t.Fatalf("expect expired token error")
	}
}
