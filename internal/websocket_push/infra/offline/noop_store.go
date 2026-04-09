package offline

import (
	"context"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type NoopStore struct{}

func NewNoopStore() *NoopStore { return &NoopStore{} }

func (s *NoopStore) Save(ctx context.Context, msg domain.PushMessage) error {
	_ = ctx
	_ = msg
	return nil
}

func (s *NoopStore) PullByUser(ctx context.Context, userID int64, limit int) ([]domain.PushMessage, error) {
	_ = ctx
	_ = userID
	_ = limit
	return []domain.PushMessage{}, nil
}
