package contextx

import (
	"context"

	"go-baseline-skeleton/internal/identity/domain"
)

type principalKey struct{}

type PrincipalStore struct{}

func NewPrincipalStore() *PrincipalStore { return &PrincipalStore{} }

func (s *PrincipalStore) WithPrincipal(ctx context.Context, p domain.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func (s *PrincipalStore) Principal(ctx context.Context) (domain.Principal, bool) {
	v := ctx.Value(principalKey{})
	p, ok := v.(domain.Principal)
	return p, ok
}