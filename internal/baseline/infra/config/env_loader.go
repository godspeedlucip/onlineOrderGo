package config

import (
	"context"
	"os"
	"strconv"
	"strings"

	"go-baseline-skeleton/internal/baseline/domain"
)

type EnvLoader struct{}

func NewEnvLoader() *EnvLoader {
	return &EnvLoader{}
}

func (l *EnvLoader) Load(ctx context.Context) (*domain.Config, error) {
	_ = ctx

	ttlSecond, err := envInt("IDEMPOTENCY_TTL_SECOND", 300)
	if err != nil {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid IDEMPOTENCY_TTL_SECOND", err)
	}

	cfg := &domain.Config{
		App: domain.AppConfig{
			Name: readOrDefault("APP_NAME", "sky-takeout-go"),
			Env:  readOrDefault("APP_ENV", "dev"),
		},
		HTTP: domain.HTTPConfig{
			Addr: readOrDefault("HTTP_ADDR", ":8080"),
		},
		Log: domain.LogConfig{
			Level: strings.ToLower(readOrDefault("LOG_LEVEL", "info")),
		},
		Idempotency: domain.IdempotencyConfig{
			Enabled:   envBool("IDEMPOTENCY_ENABLED", true),
			TTLSecond: ttlSecond,
		},
	}

	return cfg, nil
}

func readOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}