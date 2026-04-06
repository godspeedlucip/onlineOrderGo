package verify

import (
	"context"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type SimpleGrayPolicy struct {
	Enabled bool
}

func NewSimpleGrayPolicy(enabled bool) *SimpleGrayPolicy {
	return &SimpleGrayPolicy{Enabled: enabled}
}

func (p *SimpleGrayPolicy) Decide(ctx context.Context, cb *domain.VerifiedCallback) (domain.GrayDecision, error) {
	_ = ctx
	_ = cb
	return domain.GrayDecision{Enabled: p.Enabled, Reason: "static-flag"}, nil
}