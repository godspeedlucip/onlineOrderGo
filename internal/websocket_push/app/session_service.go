package app

import (
	"context"
	"fmt"
	"time"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

func (s *Service) Connect(ctx context.Context, req domain.ConnectRequest) (*domain.Session, error) {
	if s.deps.Registry == nil || s.deps.Auth == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "connect deps not initialized", nil)
	}
	if req.Token == "" {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "empty token", nil)
	}
	uid, err := s.deps.Auth.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "invalid token", err)
	}
	if req.UserID > 0 && req.UserID != uid {
		return nil, domain.NewBizError(domain.CodeUnauthorized, "token user mismatch", nil)
	}
	now := time.Now()
	session := &domain.Session{
		SessionID:   buildSessionID(uid, req.ClientType, now),
		UserID:      uid,
		ShopID:      req.ShopID,
		ClientType:  req.ClientType,
		Channels:    normalizeChannels(req.Channels),
		ConnectedAt: now,
		LastSeenAt:  now,
	}

	existings, fErr := s.deps.Registry.FindByUser(ctx, uid)
	if fErr != nil {
		return nil, fErr
	}
	for _, old := range existings {
		if old.ClientType != req.ClientType {
			continue
		}
		_ = s.Disconnect(ctx, old.SessionID)
	}

	if err := s.deps.Registry.Add(ctx, *session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) Disconnect(ctx context.Context, sessionID string) error {
	if s.deps.Registry == nil {
		return domain.NewBizError(domain.CodeInternal, "registry not initialized", nil)
	}
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	current, err := s.deps.Registry.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	if s.deps.Gateway != nil {
		_ = s.deps.Gateway.Close(ctx, sessionID)
	}
	return s.deps.Registry.Remove(ctx, sessionID)
}

func (s *Service) Heartbeat(ctx context.Context, sessionID string) error {
	if s.deps.Registry == nil {
		return domain.NewBizError(domain.CodeInternal, "registry not initialized", nil)
	}
	if sessionID == "" {
		return domain.NewBizError(domain.CodeInvalidArgument, "empty sessionId", nil)
	}
	current, err := s.deps.Registry.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if current == nil {
		return domain.NewBizError(domain.CodeConflict, "session not found", nil)
	}
	return s.deps.Registry.Touch(ctx, sessionID)
}

func buildSessionID(userID int64, clientType domain.ClientType, t time.Time) string {
	return fmt.Sprintf("%s-%s-%d-%d", t.Format("20060102150405.000"), clientType, userID, t.UnixNano())
}

func normalizeChannels(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, ch := range in {
		if ch == "" {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		out = append(out, ch)
	}
	return out
}
