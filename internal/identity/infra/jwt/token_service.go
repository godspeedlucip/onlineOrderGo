package jwt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type TokenService struct {
	issuer string
	secret []byte
}

func NewTokenService(issuer, secret string) *TokenService {
	return &TokenService{issuer: issuer, secret: []byte(secret)}
}

func (s *TokenService) Issue(ctx context.Context, claims domain.Claims) (string, time.Time, error) {
	_ = ctx
	if len(s.secret) == 0 {
		return "", time.Time{}, domain.NewBizError(domain.CodeInternal, "jwt secret is empty", nil)
	}

	headerJSON, err := json.Marshal(map[string]any{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"iss": s.issuer,
		"sub": claims.AccountID,
		"typ": string(claims.AccountType),
		"jti": claims.TokenID,
		"ver": claims.TokenVersion,
		"iat": claims.IssuedAt.Unix(),
		"exp": claims.ExpiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	head := base64.RawURLEncoding.EncodeToString(headerJSON)
	pay := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := head + "." + pay
	sig := sign(unsigned, s.secret)

	// TODO: if Java side uses different claim keys or issuer rules, align here.
	return unsigned + "." + sig, claims.ExpiresAt, nil
}

func (s *TokenService) Parse(ctx context.Context, token string) (*domain.Claims, error) {
	_ = ctx
	if len(s.secret) == 0 {
		return nil, domain.NewBizError(domain.CodeInternal, "jwt secret is empty", nil)
	}

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt format")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := sign(unsigned, s.secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("invalid jwt signature")
	}

	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(hdrBytes, &header); err != nil {
		return nil, err
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unexpected alg: %s", header.Alg)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var payload struct {
		Iss string `json:"iss"`
		Sub int64  `json:"sub"`
		Typ string `json:"typ"`
		JTI string `json:"jti"`
		Ver int64  `json:"ver"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	if s.issuer != "" && payload.Iss != s.issuer {
		return nil, errors.New("invalid issuer")
	}
	if time.Now().Unix() >= payload.Exp {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "token expired", nil)
	}

	return &domain.Claims{
		AccountID:    payload.Sub,
		AccountType:  domain.AccountType(payload.Typ),
		TokenID:      payload.JTI,
		TokenVersion: payload.Ver,
		IssuedAt:     time.Unix(payload.Iat, 0),
		ExpiresAt:    time.Unix(payload.Exp, 0),
	}, nil
}

func sign(unsigned string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}