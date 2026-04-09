package ws

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"go-baseline-skeleton/internal/websocket_push/domain"
)

type SessionConnector interface {
	Connect(ctx context.Context, req domain.ConnectRequest) (*domain.Session, error)
	Disconnect(ctx context.Context, sessionID string) error
	Heartbeat(ctx context.Context, sessionID string) error
}

type Server struct {
	connector SessionConnector
	gateway   *Gateway
	upgrader  websocket.Upgrader
}

func NewServer(connector SessionConnector, gateway *Gateway) *Server {
	return &Server{
		connector: connector,
		gateway:   gateway,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := parseToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	req := domain.ConnectRequest{
		Token:      token,
		ClientType: domain.ClientType(r.URL.Query().Get("clientType")),
		UserID:     parseInt64(r.URL.Query().Get("userId")),
		ShopID:     parseInt64(r.URL.Query().Get("shopId")),
		Channels:   parseCSV(r.URL.Query().Get("channels")),
		RemoteIP:   r.RemoteAddr,
	}
	session, err := s.connector.Connect(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), mapStatus(err))
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = s.connector.Disconnect(r.Context(), session.SessionID)
		return
	}
	s.gateway.Attach(
		session.SessionID,
		conn,
		func() {
			_ = s.connector.Heartbeat(context.Background(), session.SessionID)
		},
		func() {
			_ = s.connector.Disconnect(context.Background(), session.SessionID)
		},
	)
}

func parseToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		val := strings.TrimSpace(part)
		if val == "" {
			continue
		}
		out = append(out, val)
	}
	return out
}

func parseInt64(raw string) int64 {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func mapStatus(err error) int {
	bizErr, ok := err.(*domain.BizError)
	if !ok {
		return http.StatusInternalServerError
	}
	switch bizErr.Code {
	case domain.CodeUnauthorized:
		return http.StatusUnauthorized
	case domain.CodeInvalidArgument:
		return http.StatusBadRequest
	case domain.CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
