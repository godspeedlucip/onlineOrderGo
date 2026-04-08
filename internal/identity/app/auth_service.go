package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type AuthDeps struct {
	Repo         domain.AccountRepository
	Token        domain.TokenService
	Password     domain.PasswordService
	PrincipalCtx domain.PrincipalContext
	Tx           domain.TxManager
	Sessions     domain.SessionStore

	// Optional cross-domain dependencies. Keep injected for future expansion.
	Cache     domain.CachePort
	MQ        domain.MQPort
	WebSocket domain.WebSocketPort
	Payment   domain.PaymentPort

	// When true, logout upgrades token version to invalidate all active tokens.
	RevokeAllOnLogout bool
}

type AuthService struct {
	deps AuthDeps
}

func NewAuthService(deps AuthDeps) *AuthService {
	return &AuthService{deps: deps}
}

func (s *AuthService) Login(ctx context.Context, in domain.LoginInput) (*domain.LoginOutput, error) {
	var out *domain.LoginOutput
	run := func(runCtx context.Context) error {
		result, err := s.loginCore(runCtx, in)
		if err != nil {
			return err
		}
		out = result
		return nil
	}

	if s.deps.Tx != nil {
		if err := s.deps.Tx.RunInTx(ctx, run); err != nil {
			return nil, err
		}
		return out, nil
	}

	if err := run(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AuthService) loginCore(ctx context.Context, in domain.LoginInput) (*domain.LoginOutput, error) {
	if strings.TrimSpace(in.Identifier) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "identifier and password are required", nil)
	}
	if in.AccountType == "" {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "accountType is required", nil)
	}
	if s.deps.Repo == nil || s.deps.Token == nil || s.deps.Password == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "auth dependencies not initialized", nil)
	}

	account, err := s.deps.Repo.FindByIdentifier(ctx, in.AccountType, in.Identifier)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "invalid credentials", nil)
	}
	if account.Status != domain.AccountStatusEnabled {
		return nil, domain.NewBizError(domain.CodeForbidden, "account disabled", nil)
	}
	if err := s.deps.Password.Compare(account.PasswordHash, in.Password); err != nil {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "invalid credentials", err)
	}

	tokenVersion := int64(1)
	if s.deps.Sessions != nil {
		ver, sessionErr := s.deps.Sessions.EnsureVersion(ctx, account.Type, account.ID)
		if sessionErr != nil {
			return nil, domain.NewBizError(domain.CodeInternal, "session store unavailable", sessionErr)
		}
		tokenVersion = ver
	}

	now := time.Now()
	tokenID, genErr := generateTokenID()
	if genErr != nil {
		return nil, domain.NewBizError(domain.CodeInternal, "token id generation failed", genErr)
	}
	claims := domain.Claims{
		AccountID:    account.ID,
		AccountType:  account.Type,
		TokenID:      tokenID,
		TokenVersion: tokenVersion,
		IssuedAt:     now,
	}
	token, expireAt, err := s.deps.Token.Issue(ctx, claims)
	if err != nil {
		return nil, err
	}

	return &domain.LoginOutput{
		Token:     token,
		ExpiresAt: expireAt,
		Principal: domain.Principal{
			AccountID:    account.ID,
			AccountType:  account.Type,
			DisplayName:  account.DisplayName,
			TokenID:      tokenID,
			TokenVersion: tokenVersion,
		},
	}, nil
}

func (s *AuthService) VerifyToken(ctx context.Context, rawToken string) (domain.Principal, error) {
	if strings.TrimSpace(rawToken) == "" {
		return domain.Principal{}, domain.NewBizError(domain.CodeUnauthorized, "missing token", nil)
	}
	if s.deps.Token == nil || s.deps.Repo == nil {
		return domain.Principal{}, domain.NewBizError(domain.CodeInternal, "auth dependencies not initialized", nil)
	}

	claims, err := s.deps.Token.Parse(ctx, rawToken)
	if err != nil {
		var bizErr *domain.BizError
		if errors.As(err, &bizErr) {
			return domain.Principal{}, bizErr
		}
		return domain.Principal{}, domain.NewBizError(domain.CodeUnauthorized, "invalid token", err)
	}

	if s.deps.Sessions != nil && claims.TokenID != "" {
		revoked, revokedErr := s.deps.Sessions.IsTokenRevoked(ctx, claims.TokenID)
		if revokedErr != nil {
			return domain.Principal{}, domain.NewBizError(domain.CodeInternal, "session store unavailable", revokedErr)
		}
		if revoked {
			return domain.Principal{}, domain.NewBizError(domain.CodeUnauthorized, "token revoked", nil)
		}
		currentVersion, exists, verErr := s.deps.Sessions.GetVersion(ctx, claims.AccountType, claims.AccountID)
		if verErr != nil {
			return domain.Principal{}, domain.NewBizError(domain.CodeInternal, "session version read failed", verErr)
		}
		if exists && currentVersion != claims.TokenVersion {
			return domain.Principal{}, domain.NewBizError(domain.CodeUnauthorized, "token version expired", nil)
		}
	}

	account, err := s.deps.Repo.FindByID(ctx, claims.AccountType, claims.AccountID)
	if err != nil {
		return domain.Principal{}, err
	}
	if account == nil || account.Status != domain.AccountStatusEnabled {
		return domain.Principal{}, domain.NewBizError(domain.CodeForbidden, "account disabled", nil)
	}

	principal := domain.Principal{
		AccountID:    account.ID,
		AccountType:  account.Type,
		DisplayName:  account.DisplayName,
		TokenID:      claims.TokenID,
		TokenVersion: claims.TokenVersion,
	}
	return principal, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return domain.NewBizError(domain.CodeUnauthorized, "missing token", nil)
	}
	if s.deps.Token == nil {
		return domain.NewBizError(domain.CodeInternal, "token service not initialized", nil)
	}
	if s.deps.Sessions == nil {
		return nil
	}

	claims, err := s.deps.Token.Parse(ctx, rawToken)
	if err != nil {
		return domain.NewBizError(domain.CodeUnauthorized, "invalid token", err)
	}

	if claims.TokenID != "" {
		revoked, checkErr := s.deps.Sessions.IsTokenRevoked(ctx, claims.TokenID)
		if checkErr != nil {
			return domain.NewBizError(domain.CodeInternal, "session store unavailable", checkErr)
		}
		if revoked {
			return nil
		}
		if revokeErr := s.deps.Sessions.MarkTokenRevoked(ctx, claims.TokenID, claims.ExpiresAt); revokeErr != nil {
			return domain.NewBizError(domain.CodeInternal, "token revoke failed", revokeErr)
		}
	}

	if s.deps.RevokeAllOnLogout {
		_, updated, casErr := s.deps.Sessions.CompareAndIncreaseVersion(ctx, claims.AccountType, claims.AccountID, claims.TokenVersion)
		if casErr != nil {
			return domain.NewBizError(domain.CodeInternal, "session version update failed", casErr)
		}
		if !updated {
			return nil
		}
	}
	return nil
}

func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
