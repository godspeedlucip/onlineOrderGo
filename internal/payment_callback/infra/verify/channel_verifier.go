package verify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-baseline-skeleton/internal/payment_callback/domain"
)

type ChannelVerifier struct {
	secrets map[string]string
	nowFn   func() time.Time
}

func NewChannelVerifier(channelSecrets map[string]string) *ChannelVerifier {
	normalized := make(map[string]string, len(channelSecrets))
	for k, v := range channelSecrets {
		key := strings.ToUpper(strings.TrimSpace(k))
		secret := strings.TrimSpace(v)
		if key == "" || secret == "" {
			continue
		}
		normalized[key] = secret
	}
	return &ChannelVerifier{
		secrets: normalized,
		nowFn:   time.Now,
	}
}

func (v *ChannelVerifier) VerifyAndParse(ctx context.Context, headers map[string]string, body []byte) (*domain.VerifiedCallback, error) {
	_ = ctx
	var payload struct {
		NotifyID      string `json:"notifyId"`
		OrderNo       string `json:"orderNo"`
		TransactionNo string `json:"transactionNo"`
		PaidAmount    int64  `json:"paidAmount"`
		PaidAt        string `json:"paidAt"`
		Status        string `json:"status"`
		MerchantID    string `json:"merchantId"`
		Channel       string `json:"channel"`
		Signature     string `json:"signature"`
		Timestamp     string `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	channel := strings.ToUpper(strings.TrimSpace(firstNonEmpty(
		headers["X-Pay-Channel"], headers["x-pay-channel"], payload.Channel,
	)))
	if channel == "" {
		return nil, errors.New("missing callback channel")
	}
	secret, ok := v.secrets[channel]
	if !ok || strings.TrimSpace(secret) == "" {
		return nil, errors.New("missing channel secret")
	}

	signature := strings.TrimSpace(firstNonEmpty(
		headers["X-Pay-Signature"], headers["x-pay-signature"], payload.Signature,
	))
	timestamp := strings.TrimSpace(firstNonEmpty(
		headers["X-Pay-Timestamp"], headers["x-pay-timestamp"], payload.Timestamp,
	))
	if signature == "" || timestamp == "" {
		return nil, errors.New("missing signature headers")
	}
	if !verifyHMACSHA256(secret, timestamp, body, signature) {
		return nil, errors.New("invalid callback signature")
	}

	if strings.TrimSpace(payload.OrderNo) == "" || strings.TrimSpace(payload.TransactionNo) == "" {
		return nil, errors.New("missing required callback fields")
	}
	if payload.PaidAmount <= 0 {
		return nil, errors.New("invalid paidAmount")
	}

	paidAt := parsePaidAt(payload.PaidAt, v.nowFn)
	return &domain.VerifiedCallback{
		Channel:       channel,
		NotifyID:      payload.NotifyID,
		OrderNo:       payload.OrderNo,
		TransactionNo: payload.TransactionNo,
		PaidAmount:    payload.PaidAmount,
		PaidAt:        paidAt,
		RawStatus:     payload.Status,
		MerchantID:    payload.MerchantID,
	}, nil
}

func verifyHMACSHA256(secret, timestamp string, body []byte, provided string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	provided = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(provided), "sha256="))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func parsePaidAt(raw string, nowFn func() time.Time) time.Time {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nowFn()
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
		return t
	}
	return nowFn()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
