package jwt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-baseline-skeleton/internal/identity/domain"
)

type AccountJWTConfig struct {
	Issuer   string
	Secret   string
	TTL      time.Duration
	ClaimKey string
}

type Config struct {
	Employee  AccountJWTConfig
	User      AccountJWTConfig
	Algorithm string
}

type TokenService struct {
	cfg Config
}

func NewTokenService(cfg Config) *TokenService {
	if strings.TrimSpace(cfg.Algorithm) == "" {
		cfg.Algorithm = "HS256"
	}
	if strings.TrimSpace(cfg.Employee.ClaimKey) == "" {
		cfg.Employee.ClaimKey = "empId"
	}
	if strings.TrimSpace(cfg.User.ClaimKey) == "" {
		cfg.User.ClaimKey = "userId"
	}
	if cfg.Employee.TTL <= 0 {
		cfg.Employee.TTL = 2 * time.Hour
	}
	if cfg.User.TTL <= 0 {
		cfg.User.TTL = 2 * time.Hour
	}
	return &TokenService{cfg: cfg}
}

func (s *TokenService) Issue(ctx context.Context, claims domain.Claims) (string, time.Time, error) {
	_ = ctx

	accountCfg, err := s.accountConfig(claims.AccountType)
	if err != nil {
		return "", time.Time{}, err
	}
	if strings.TrimSpace(accountCfg.Secret) == "" {
		return "", time.Time{}, domain.NewBizError(domain.CodeInternal, "jwt secret is empty", nil)
	}

	issuedAt := claims.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = time.Now()
	}
	expiresAt := issuedAt.Add(accountCfg.TTL)

	headerJSON, err := json.Marshal(map[string]any{"alg": s.cfg.Algorithm, "typ": "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}

	payload := map[string]any{
		accountCfg.ClaimKey: claims.AccountID,
		"jti":             claims.TokenID,
		"ver":             claims.TokenVersion,
		"iat":             issuedAt.Unix(),
		"exp":             expiresAt.Unix(),
	}
	if strings.TrimSpace(accountCfg.Issuer) != "" {
		payload["iss"] = accountCfg.Issuer
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, err
	}

	head := base64.RawURLEncoding.EncodeToString(headerJSON)
	pay := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := head + "." + pay
	sig := sign(unsigned, []byte(accountCfg.Secret))
	return unsigned + "." + sig, expiresAt, nil
}

func (s *TokenService) Parse(ctx context.Context, token string) (*domain.Claims, error) {
	_ = ctx

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt format")
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
	if !strings.EqualFold(header.Alg, s.cfg.Algorithm) {
		return nil, fmt.Errorf("unexpected alg: %s", header.Alg)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	accountType, accountID, cfg, err := s.resolveAccountFromPayload(payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, domain.NewBizError(domain.CodeInternal, "jwt secret is empty", nil)
	}

	unsigned := parts[0] + "." + parts[1]
	expected := sign(unsigned, []byte(cfg.Secret))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("invalid jwt signature")
	}

	if cfg.Issuer != "" {
		iss, _ := payload["iss"].(string)
		if iss != cfg.Issuer {
			return nil, errors.New("invalid issuer")
		}
	}

	exp, err := toInt64(payload["exp"])
	if err != nil {
		return nil, errors.New("invalid exp")
	}
	if time.Now().Unix() >= exp {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "token expired", nil)
	}

	iat, _ := toInt64(payload["iat"])
	ver, _ := toInt64(payload["ver"])
	jti, _ := payload["jti"].(string)

	return &domain.Claims{
		AccountID:    accountID,
		AccountType:  accountType,
		TokenID:      jti,
		TokenVersion: ver,
		IssuedAt:     unixOrZero(iat),
		ExpiresAt:    time.Unix(exp, 0),
	}, nil
}

func (s *TokenService) accountConfig(accountType domain.AccountType) (AccountJWTConfig, error) {
	switch accountType {
	case domain.AccountTypeEmployee:
		return s.cfg.Employee, nil
	case domain.AccountTypeUser:
		return s.cfg.User, nil
	default:
		return AccountJWTConfig{}, domain.NewBizError(domain.CodeInvalidArgument, "unsupported account type", nil)
	}
}

func (s *TokenService) resolveAccountFromPayload(payload map[string]any) (domain.AccountType, int64, AccountJWTConfig, error) {
	if raw, ok := payload[s.cfg.Employee.ClaimKey]; ok {
		id, err := toInt64(raw)
		if err == nil {
			return domain.AccountTypeEmployee, id, s.cfg.Employee, nil
		}
	}
	if raw, ok := payload[s.cfg.User.ClaimKey]; ok {
		id, err := toInt64(raw)
		if err == nil {
			return domain.AccountTypeUser, id, s.cfg.User, nil
		}
	}

	// Backward compatibility for earlier Go skeleton tokens.
	typ, _ := payload["typ"].(string)
	sub, err := toInt64(payload["sub"])
	if err == nil {
		accountType := domain.AccountType(typ)
		cfg, cfgErr := s.accountConfig(accountType)
		if cfgErr == nil {
			return accountType, sub, cfg, nil
		}
	}

	return "", 0, AccountJWTConfig{}, errors.New("invalid jwt claims")
}

func sign(unsigned string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func toInt64(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case json.Number:
		return t.Int64()
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, errors.New("not a number")
	}
}

func unixOrZero(ts int64) time.Time {
	if ts <= 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}
