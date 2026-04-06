package payment

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

type Gateway struct{}

func NewGateway() *Gateway { return &Gateway{} }

func (g *Gateway) PreparePayment(ctx context.Context, req domain.PaymentRequest) (*domain.PaymentResponse, error) {
	_ = ctx
	_ = req
	// TODO: integrate payment channel prepay API.
	return &domain.PaymentResponse{PrepayToken: ""}, nil
}
