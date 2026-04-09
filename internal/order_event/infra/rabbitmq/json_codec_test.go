package rabbitmq

import (
	"testing"

	"go-baseline-skeleton/internal/order_event/domain"
)

func TestJSONCodec_EncodeDecode(t *testing.T) {
	codec := NewJSONCodec()
	evt := domain.OrderEvent{EventID: "E1", EventType: domain.EventOrderCreated, OrderID: 1, OrderNo: "O1", Version: 1}
	body, headers, err := codec.Encode(evt)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	out, err := codec.Decode(domain.ConsumeMessage{MessageID: evt.EventID, Headers: headers, Body: body})
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.EventID != evt.EventID || out.EventType != evt.EventType || out.OrderID != evt.OrderID {
		t.Fatalf("unexpected decoded event: %+v", out)
	}
}

func TestReadRetryCount(t *testing.T) {
	if n := readRetryCount(nil); n != 0 {
		t.Fatalf("retry count from nil should be 0, got=%d", n)
	}
}
