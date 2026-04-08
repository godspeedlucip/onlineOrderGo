package contextx

import (
	"context"

	"go-baseline-skeleton/internal/identity/domain"
)

type principalKey struct{}
type userIDKey struct{}
type roleKey struct{}

type PrincipalStore struct{}

func NewPrincipalStore() *PrincipalStore { return &PrincipalStore{} }

func (s *PrincipalStore) WithPrincipal(ctx context.Context, p domain.Principal) context.Context {
	ctx = context.WithValue(ctx, principalKey{}, p)
	ctx = context.WithValue(ctx, userIDKey{}, p.AccountID)
	ctx = context.WithValue(ctx, roleKey{}, string(p.AccountType))
	return ctx
}

func (s *PrincipalStore) Principal(ctx context.Context) (domain.Principal, bool) {
	v := ctx.Value(principalKey{})
	p, ok := v.(domain.Principal)
	return p, ok
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDKey{})
	id, ok := v.(int64)
	return id, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(roleKey{})
	role, ok := v.(string)
	return role, ok
}
