package domain

import "time"

type OrderEventType string

const (
	OrderEventCreated       OrderEventType = "ORDER_CREATED"
	OrderEventCanceled      OrderEventType = "ORDER_CANCELED"
	OrderEventStatusChanged OrderEventType = "ORDER_STATUS_CHANGED"
)

type OrderEvent struct {
	Type      OrderEventType
	OrderID   int64
	OrderNo   string
	From      OrderStatus
	To        OrderStatus
	OccurredAt time.Time
}
