package domain

import "time"

type ClientType string

const (
	ClientTypeUser     ClientType = "USER"
	ClientTypeEmployee ClientType = "EMPLOYEE"
)

type ConnectRequest struct {
	Token      string
	ClientType ClientType
	UserID     int64
	ShopID     int64
	Channels   []string
	RemoteIP   string
}

type Session struct {
	SessionID   string
	UserID      int64
	ShopID      int64
	ClientType  ClientType
	Channels    []string
	ConnectedAt time.Time
	LastSeenAt  time.Time
}

type PushMessage struct {
	MessageID  string            `json:"messageId"`
	EventType  string            `json:"eventType"`
	TargetUser int64             `json:"targetUser,omitempty"`
	TargetShop int64             `json:"targetShop,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Payload    []byte            `json:"payload"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
}

type PushResult struct {
	Delivered int
	Failed    int
}
