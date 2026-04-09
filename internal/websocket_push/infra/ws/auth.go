package ws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type JWTAuthConfig struct {
	Secret    string
	Algorithm string
	Issuer    string
	Audience  string
	UserClaim string
}

type TokenAuth struct {
	secret    []byte
	algorithm string
	issuer    string
	audience  string
	userClaim string
}

func NewTokenAuth(cfg JWTAuthConfig) *TokenAuth {
	algo := strings.TrimSpace(cfg.Algorithm)
	if algo == "" {
		algo = jwt.SigningMethodHS256.Alg()
	}
	claim := strings.TrimSpace(cfg.UserClaim)
	if claim == "" {
		claim = "userId"
	}
	return &TokenAuth{
		secret:    []byte(cfg.Secret),
		algorithm: algo,
		issuer:    strings.TrimSpace(cfg.Issuer),
		audience:  strings.TrimSpace(cfg.Audience),
		userClaim: claim,
	}
}

func (a *TokenAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	_ = ctx
	raw := strings.TrimSpace(token)
	if raw == "" {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "empty token", nil)
	}

	parser := jwt.NewParser(
		buildParserOptions(a)...,
	)

	claims := jwt.MapClaims{}
	parsed, err := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != a.algorithm {
			return nil, fmt.Errorf("unexpected signing alg %s", t.Method.Alg())
		}
		return a.secret, nil
	})
	if err != nil || !parsed.Valid {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "invalid token", err)
	}

	value, ok := claims[a.userClaim]
	if !ok {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "missing user claim", nil)
	}
	userID, convErr := claimToInt64(value)
	if convErr != nil || userID <= 0 {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "invalid user claim", convErr)
	}
	return userID, nil
}

func claimToInt64(v any) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported claim type %T", v)
	}
}

func buildParserOptions(a *TokenAuth) []jwt.ParserOption {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{a.algorithm}),
		jwt.WithExpirationRequired(),
	}
	if a.issuer != "" {
		opts = append(opts, jwt.WithIssuer(a.issuer))
	}
	if a.audience != "" {
		opts = append(opts, jwt.WithAudience(a.audience))
	}
	return opts
}
