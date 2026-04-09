package app

import (
	"context"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

func (s *Service) Push(ctx context.Context, msg domain.PushMessage) (*domain.PushResult, error) {
	if s.deps.Registry == nil || s.deps.Gateway == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "push deps not initialized", nil)
	}
	if len(msg.Payload) == 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty payload", nil)
	}
	if msg.MessageID == "" {
		msg.MessageID = "msg-" + time.Now().Format("20060102150405.000")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	duplicated, err := s.isDuplicateMessage(ctx, msg.MessageID)
	if err != nil {
		return nil, err
	}
	if duplicated {
		return &domain.PushResult{Delivered: 0, Failed: 0}, nil
	}

	targets, err := s.resolveTargets(ctx, msg)
	if err != nil {
		return nil, err
	}
	res := &domain.PushResult{}
	for _, session := range targets {
		if err := s.deps.Gateway.Send(ctx, session.SessionID, msg.Payload); err != nil {
			res.Failed++
			continue
		}
		res.Delivered++
	}
	if res.Delivered == 0 && s.deps.MQ != nil {
		if s.deps.Offline != nil {
			_ = s.deps.Offline.Save(ctx, msg)
		}
		_ = s.deps.MQ.PublishBroadcast(ctx, msg)
	}
	return res, nil
}

func (s *Service) Broadcast(ctx context.Context, msg domain.PushMessage) (*domain.PushResult, error) {
	if s.deps.Registry == nil || s.deps.Gateway == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "broadcast deps not initialized", nil)
	}
	if len(msg.Payload) == 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty payload", nil)
	}
	if msg.MessageID == "" {
		msg.MessageID = "broadcast-" + time.Now().Format("20060102150405.000")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	duplicated, err := s.isDuplicateMessage(ctx, msg.MessageID)
	if err != nil {
		return nil, err
	}
	if duplicated {
		return &domain.PushResult{Delivered: 0, Failed: 0}, nil
	}

	sessions, err := s.deps.Registry.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	res := &domain.PushResult{}
	for _, session := range sessions {
		if err := s.deps.Gateway.Send(ctx, session.SessionID, msg.Payload); err != nil {
			res.Failed++
			continue
		}
		res.Delivered++
	}
	if s.deps.MQ != nil {
		_ = s.deps.MQ.PublishBroadcast(ctx, msg)
	}
	return res, nil
}

func (s *Service) PullOffline(ctx context.Context, userID int64, limit int) ([]domain.PushMessage, error) {
	if userID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid userId", nil)
	}
	if limit <= 0 {
		limit = 50
	}
	if s.deps.Offline == nil {
		return []domain.PushMessage{}, nil
	}
	return s.deps.Offline.PullByUser(ctx, userID, limit)
}

func (s *Service) resolveTargets(ctx context.Context, msg domain.PushMessage) ([]domain.Session, error) {
	if msg.TargetUser > 0 {
		return s.deps.Registry.FindByUser(ctx, msg.TargetUser)
	}
	if msg.TargetShop > 0 {
		return s.deps.Registry.FindByShop(ctx, msg.TargetShop)
	}
	if msg.Channel != "" {
		return s.deps.Registry.FindByChannel(ctx, msg.Channel)
	}
	return nil, domain.NewBizError(domain.CodeInvalidArgument, "missing push target", nil)
}

func (s *Service) DeliverLocal(ctx context.Context, msg domain.PushMessage) (*domain.PushResult, error) {
	if s.deps.Registry == nil || s.deps.Gateway == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "broadcast deps not initialized", nil)
	}
	if len(msg.Payload) == 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty payload", nil)
	}
	if msg.MessageID == "" {
		msg.MessageID = "broadcast-" + time.Now().Format("20060102150405.000")
	}
	duplicated, err := s.isDuplicateMessage(ctx, msg.MessageID)
	if err != nil {
		return nil, err
	}
	if duplicated {
		return &domain.PushResult{Delivered: 0, Failed: 0}, nil
	}
	sessions, err := s.deps.Registry.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	res := &domain.PushResult{}
	for _, session := range sessions {
		if err := s.deps.Gateway.Send(ctx, session.SessionID, msg.Payload); err != nil {
			res.Failed++
			continue
		}
		res.Delivered++
	}
	return res, nil
}

func (s *Service) isDuplicateMessage(ctx context.Context, messageID string) (bool, error) {
	if messageID == "" {
		return false, domain.NewBizError(domain.CodeInvalidArgument, "empty messageId", nil)
	}
	if s.deps.Dedup == nil {
		return false, nil
	}
	acquired, err := s.deps.Dedup.TryAcquire(ctx, messageID, s.deps.PushDedupTTL)
	if err != nil {
		return false, domain.NewBizError(domain.CodeInternal, "dedupe store error", err)
	}
	return !acquired, nil
}
