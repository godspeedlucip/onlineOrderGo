package rabbitmq

import (
	"encoding/json"

	"go-baseline-skeleton/internal/order_event/domain"
)

type JSONCodec struct{}

func NewJSONCodec() *JSONCodec { return &JSONCodec{} }

func (c *JSONCodec) Encode(evt domain.OrderEvent) ([]byte, map[string]string, error) {
	body, err := json.Marshal(evt)
	if err != nil {
		return nil, nil, err
	}
	headers := map[string]string{
		"eventType": string(evt.EventType),
		"version":   "1",
	}
	return body, headers, nil
}

func (c *JSONCodec) Decode(msg domain.ConsumeMessage) (*domain.OrderEvent, error) {
	if len(msg.Body) == 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "empty message body", nil)
	}
	var evt domain.OrderEvent
	if err := json.Unmarshal(msg.Body, &evt); err != nil {
		return nil, err
	}
	if evt.EventID == "" {
		evt.EventID = msg.MessageID
	}
	return &evt, nil
}
