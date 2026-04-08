package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go-baseline-skeleton/internal/identity/domain"
)

type SQLAccountRepo struct {
	db *sql.DB
}

func NewSQLAccountRepo(db *sql.DB) *SQLAccountRepo {
	return &SQLAccountRepo{db: db}
}

func (r *SQLAccountRepo) FindByIdentifier(ctx context.Context, accountType domain.AccountType, identifier string) (*domain.Account, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "identity db is not initialized", nil)
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, nil
	}

	switch accountType {
	case domain.AccountTypeEmployee:
		const q = `select id, username, name, password, status from employee where username = ? limit 1`
		return r.queryEmployeeBy(ctx, q, identifier)
	case domain.AccountTypeUser:
		const q = `select id, openid, name from user where openid = ? limit 1`
		return r.queryUserBy(ctx, q, identifier)
	default:
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "unsupported account type", nil)
	}
}

func (r *SQLAccountRepo) FindByID(ctx context.Context, accountType domain.AccountType, id int64) (*domain.Account, error) {
	if r == nil || r.db == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "identity db is not initialized", nil)
	}

	switch accountType {
	case domain.AccountTypeEmployee:
		const q = `select id, username, name, password, status from employee where id = ? limit 1`
		return r.queryEmployeeBy(ctx, q, id)
	case domain.AccountTypeUser:
		const q = `select id, openid, name from user where id = ? limit 1`
		return r.queryUserBy(ctx, q, id)
	default:
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "unsupported account type", nil)
	}
}

func (r *SQLAccountRepo) queryEmployeeBy(ctx context.Context, query string, arg any) (*domain.Account, error) {
	var (
		id       int64
		username string
		name     string
		password string
		status   int
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&id, &username, &name, &password, &status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query employee failed: %w", err)
	}

	mappedStatus := domain.AccountStatusDisabled
	if status == 1 {
		mappedStatus = domain.AccountStatusEnabled
	}

	return &domain.Account{
		ID:           id,
		Type:         domain.AccountTypeEmployee,
		Username:     username,
		DisplayName:  name,
		PasswordHash: password,
		Status:       mappedStatus,
	}, nil
}

func (r *SQLAccountRepo) queryUserBy(ctx context.Context, query string, arg any) (*domain.Account, error) {
	var (
		id     int64
		openid string
		name   string
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&id, &openid, &name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user failed: %w", err)
	}

	return &domain.Account{
		ID:           id,
		Type:         domain.AccountTypeUser,
		Username:     openid,
		DisplayName:  name,
		PasswordHash: "",
		Status:       domain.AccountStatusEnabled,
	}, nil
}
