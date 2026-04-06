package mq

import (
	"context"

	"go-baseline-skeleton/internal/order/domain"
)

type Publisher struct{}

func NewPublisher() *Publisher { return &Publisher{} }

func (p *Publisher) PublishOrderEvent(ctx context.Context, evt domain.OrderEvent) error {
	_ = ctx
	_ = evt
	// TODO: implement MQ publish or outbox integration.
	return nil
}
