package ws

import "context"

type Server struct {
	Gateway *Gateway
}

func NewServer(gateway *Gateway) *Server {
	return &Server{Gateway: gateway}
}

func (s *Server) Start(ctx context.Context) error {
	_ = ctx
	// TODO: start real websocket server and register connection lifecycle callbacks.
	return nil
}
