package verify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestChannelVerifier_VerifyAndParse(t *testing.T) {
	body := []byte(`{"notifyId":"N1","orderNo":"O1","transactionNo":"T1","paidAmount":1000,"paidAt":"2026-04-08T10:00:00Z","status":"SUCCESS","merchantId":"M1","channel":"WECHAT"}`)
	timestamp := "1712540000"
	signature := sign("sec_wechat", timestamp, body)

	v := NewChannelVerifier(map[string]string{"WECHAT": "sec_wechat"})
	out, err := v.VerifyAndParse(context.Background(), map[string]string{
		"X-Pay-Channel":   "WECHAT",
		"X-Pay-Timestamp": timestamp,
		"X-Pay-Signature": signature,
	}, body)
	if err != nil {
		t.Fatalf("VerifyAndParse failed: %v", err)
	}
	if out.OrderNo != "O1" || out.TransactionNo != "T1" || out.PaidAmount != 1000 {
		t.Fatalf("unexpected parsed callback: %+v", out)
	}
}

func TestChannelVerifier_InvalidSignature(t *testing.T) {
	body := []byte(`{"notifyId":"N1","orderNo":"O1","transactionNo":"T1","paidAmount":1000,"status":"SUCCESS","merchantId":"M1","channel":"WECHAT"}`)
	v := NewChannelVerifier(map[string]string{"WECHAT": "sec_wechat"})
	_, err := v.VerifyAndParse(context.Background(), map[string]string{
		"X-Pay-Channel":   "WECHAT",
		"X-Pay-Timestamp": "1712540000",
		"X-Pay-Signature": "deadbeef",
	}, body)
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
