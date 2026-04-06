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

	if s.isDuplicateMessage(msg.MessageID) {
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
		// TODO: fallback publish for offline sessions (e.g. delayed notify).
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
	if s.isDuplicateMessage(msg.MessageID) {
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

func (s *Service) isDuplicateMessage(messageID string) bool {
	now := time.Now()
	expireBefore := now.Add(-s.deps.PushDedupTTL)

	s.dedupeMu.Lock()
	defer s.dedupeMu.Unlock()
	for k, ts := range s.dedupe {
		if ts.Before(expireBefore) {
			delete(s.dedupe, k)
		}
	}
	if _, ok := s.dedupe[messageID]; ok {
		return true
	}
	s.dedupe[messageID] = now
	return false
}
