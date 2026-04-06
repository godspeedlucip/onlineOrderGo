package ws

import (
	"context"
	"strconv"
	"strings"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type TokenAuth struct{}

func NewTokenAuth() *TokenAuth { return &TokenAuth{} }

func (a *TokenAuth) ValidateToken(ctx context.Context, token string) (int64, error) {
	_ = ctx
	if token == "" {
		return 0, domain.NewBizError(domain.CodeUnauthorized, "empty token", nil)
	}
	// Simple runnable format: "uid:<number>".
	// TODO: replace with real JWT validation and claims parsing.
	if strings.HasPrefix(token, "uid:") {
		num := strings.TrimPrefix(token, "uid:")
		id, err := strconv.ParseInt(num, 10, 64)
		if err != nil || id <= 0 {
			return 0, domain.NewBizError(domain.CodeUnauthorized, "invalid token uid", err)
		}
		return id, nil
	}
	return 1, nil
}
