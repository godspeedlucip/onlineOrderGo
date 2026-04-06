package domain

import "time"

type CallbackInput struct {
	Headers map[string]string
	Body    []byte
	RemoteIP string
}

type CallbackAck struct {
	HTTPStatus int
	Body       string
}

type VerifiedCallback struct {
	Channel         string
	NotifyID        string
	OrderNo         string
	TransactionNo   string
	PaidAmount      int64
	PaidAt          time.Time
	RawStatus       string
	MerchantID      string
}

type OrderSnapshot struct {
	OrderID       int64
	OrderNo       string
	Status        string
	TotalAmount   int64
	MerchantID    string
}

type PaymentRecord struct {
	OrderID        int64
	OrderNo        string
	TransactionNo  string
	Channel        string
	PaidAmount     int64
	PaidAt         time.Time
	RawStatus      string
}

type CallbackLog struct {
	NotifyID       string
	OrderNo        string
	TransactionNo  string
	Channel        string
	HTTPHeaders    map[string]string
	Body           []byte
	Verified       bool
	ErrorMessage   string
	CreatedAt      time.Time
}

type GrayDecision struct {
	Enabled bool
	Reason  string
}

type OrderPaidEvent struct {
	OrderID       int64
	OrderNo       string
	TransactionNo string
	PaidAmount    int64
	PaidAt        time.Time
}