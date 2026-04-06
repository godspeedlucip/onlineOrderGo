package domain

import (
	"context"
	"net/http"
	"time"
)

type AccountRepository interface {
	FindByIdentifier(ctx context.Context, accountType AccountType, identifier string) (*Account, error)
	FindByID(ctx context.Context, accountType AccountType, id int64) (*Account, error)
}

type TokenService interface {
	Issue(ctx context.Context, claims Claims) (token string, expireAt time.Time, err error)
	Parse(ctx context.Context, token string) (*Claims, error)
}

type PasswordService interface {
	Compare(hashed string, plain string) error
}

type PrincipalContext interface {
	WithPrincipal(ctx context.Context, p Principal) context.Context
	Principal(ctx context.Context) (Principal, bool)
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type SessionStore interface {
	EnsureVersion(ctx context.Context, accountType AccountType, accountID int64) (version int64, err error)
	GetVersion(ctx context.Context, accountType AccountType, accountID int64) (version int64, exists bool, err error)
	CompareAndIncreaseVersion(ctx context.Context, accountType AccountType, accountID int64, expected int64) (newVersion int64, updated bool, err error)
	MarkTokenRevoked(ctx context.Context, tokenID string, expireAt time.Time) error
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}

type CachePort interface {
	Ping(ctx context.Context) error
}

type MQPort interface {
	Ping(ctx context.Context) error
}

type WebSocketPort interface {
	Ping(ctx context.Context) error
}

type PaymentPort interface {
	Ping(ctx context.Context) error
}

type AuthUsecase interface {
	Login(ctx context.Context, in LoginInput) (*LoginOutput, error)
	VerifyToken(ctx context.Context, token string) (Principal, error)
	Logout(ctx context.Context, token string) error
}

type AuthMiddleware interface {
	RequireAuth(next http.Handler) http.Handler
}