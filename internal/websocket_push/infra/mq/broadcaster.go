package mq

import (
	"context"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type Broadcaster struct{}

func NewBroadcaster() *Broadcaster { return &Broadcaster{} }

func (b *Broadcaster) PublishBroadcast(ctx context.Context, msg domain.PushMessage) error {
	_ = ctx
	_ = msg
	// TODO: bridge to MQ/Redis pubsub for cross-node broadcast.
	return nil
}
