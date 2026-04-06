package verify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type ChannelVerifier struct{}

func NewChannelVerifier() *ChannelVerifier { return &ChannelVerifier{} }

func (v *ChannelVerifier) VerifyAndParse(ctx context.Context, headers map[string]string, body []byte) (*domain.VerifiedCallback, error) {
	_ = ctx
	_ = headers
	// TODO: replace with provider-specific signature verification.
	var payload struct {
		NotifyID      string `json:"notifyId"`
		OrderNo       string `json:"orderNo"`
		TransactionNo string `json:"transactionNo"`
		PaidAmount    int64  `json:"paidAmount"`
		PaidAt        string `json:"paidAt"`
		Status        string `json:"status"`
		MerchantID    string `json:"merchantId"`
		Channel       string `json:"channel"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.OrderNo) == "" || strings.TrimSpace(payload.TransactionNo) == "" {
		return nil, errors.New("missing required callback fields")
	}
	if payload.PaidAmount <= 0 {
		return nil, errors.New("invalid paidAmount")
	}
	paidAt, _ := time.Parse(time.RFC3339, payload.PaidAt)
	if paidAt.IsZero() {
		paidAt = time.Now()
	}
	return &domain.VerifiedCallback{
		Channel:       payload.Channel,
		NotifyID:      payload.NotifyID,
		OrderNo:       payload.OrderNo,
		TransactionNo: payload.TransactionNo,
		PaidAmount:    payload.PaidAmount,
		PaidAt:        paidAt,
		RawStatus:     payload.Status,
		MerchantID:    payload.MerchantID,
	}, nil
}
