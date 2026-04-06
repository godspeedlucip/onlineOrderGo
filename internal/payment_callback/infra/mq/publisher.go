package mq

import (
	"context"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type EventPublisher struct{}

func NewEventPublisher() *EventPublisher { return &EventPublisher{} }

func (p *EventPublisher) PublishOrderPaid(ctx context.Context, evt domain.OrderPaidEvent) error {
	_ = ctx
	_ = evt
	// TODO: implement message publish to MQ/outbox.
	return nil
}