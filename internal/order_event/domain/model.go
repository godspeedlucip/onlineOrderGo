package domain

import "time"

type EventType string

const (
	EventOrderCreated   EventType = "ORDER_CREATED"
	EventOrderCanceled  EventType = "ORDER_CANCELED"
	EventOrderStatusSet EventType = "ORDER_STATUS_CHANGED"
)

type OrderEvent struct {
	EventID      string    `json:"eventId"`
	EventType    EventType `json:"eventType"`
	OrderID      int64     `json:"orderId"`
	OrderNo      string    `json:"orderNo"`
	FromStatus   string    `json:"fromStatus,omitempty"`
	ToStatus     string    `json:"toStatus,omitempty"`
	OccurredAt   time.Time `json:"occurredAt"`
	TraceID      string    `json:"traceId,omitempty"`
	Payload      []byte    `json:"payload,omitempty"`
	Version      int       `json:"version"`
}

type ConsumeMessage struct {
	MessageID  string
	Headers    map[string]string
	Body       []byte
	ReceivedAt time.Time
}
