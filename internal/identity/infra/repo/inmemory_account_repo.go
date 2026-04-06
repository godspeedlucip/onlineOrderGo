package repo

import (
	"context"
	"strconv"
	"sync"

	"go-baseline-skeleton/internal/identity/domain"
)

type InMemoryAccountRepo struct {
	mu    sync.RWMutex
	items map[string]*domain.Account
}

func NewInMemoryAccountRepo(seed []*domain.Account) *InMemoryAccountRepo {
	m := make(map[string]*domain.Account)
	for _, a := range seed {
		if a == nil {
			continue
		}
		m[buildLookup(a.Type, a.Username)] = a
		m[buildIDLookup(a.Type, a.ID)] = a
	}
	return &InMemoryAccountRepo{items: m}
}

func (r *InMemoryAccountRepo) FindByIdentifier(ctx context.Context, accountType domain.AccountType, identifier string) (*domain.Account, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	acc, ok := r.items[buildLookup(accountType, identifier)]
	if !ok {
		return nil, nil
	}
	copy := *acc
	return &copy, nil
}

func (r *InMemoryAccountRepo) FindByID(ctx context.Context, accountType domain.AccountType, id int64) (*domain.Account, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	acc, ok := r.items[buildIDLookup(accountType, id)]
	if !ok {
		return nil, nil
	}
	copy := *acc
	return &copy, nil
}

func buildLookup(accountType domain.AccountType, identifier string) string {
	return string(accountType) + ":identifier:" + identifier
}

func buildIDLookup(accountType domain.AccountType, id int64) string {
	return string(accountType) + ":id:" + strconv.FormatInt(id, 10)
}