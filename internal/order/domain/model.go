package domain

import "time"

type OrderStatus string

const (
	OrderStatusPendingPay OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid       OrderStatus = "PAID"
	OrderStatusAccepted   OrderStatus = "ACCEPTED"
	OrderStatusDelivering OrderStatus = "DELIVERING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusCanceled   OrderStatus = "CANCELED"
)

type Order struct {
	OrderID      int64
	OrderNo      string
	UserID       int64
	Status       OrderStatus
	TotalAmount  int64
	Remark       string
	Version      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OrderItem struct {
	OrderID     int64
	ItemType    string
	SkuID       int64
	Flavor      string
	Name        string
	Quantity    int64
	UnitAmount  int64
	LineAmount  int64
}

type CreateOrderCommand struct {
	UserID         int64
	AddressID      int64
	Remark         string
	PaymentMethod  string
	IdempotencyKey string
}

type CancelOrderCommand struct {
	OrderID        int64
	OperatorID     int64
	Reason         string
	IdempotencyKey string
}

type TransitStatusCommand struct {
	OrderID        int64
	From           OrderStatus
	To             OrderStatus
	TriggerSource  string
	IdempotencyKey string
}

type OrderView struct {
	OrderID     int64       `json:"orderId"`
	OrderNo     string      `json:"orderNo"`
	Status      OrderStatus `json:"status"`
	TotalAmount int64       `json:"totalAmount"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type PaymentRequest struct {
	OrderID       int64
	OrderNo       string
	Amount        int64
	PaymentMethod string
}

type PaymentResponse struct {
	PrepayToken string
}
