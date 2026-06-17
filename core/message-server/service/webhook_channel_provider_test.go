package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"
	"time"
)

func TestDingTalkWebhookSignatureUsesTimestampAndSecret(t *testing.T) {
	provider := NewDingTalkWebhookChannelProvider(time.Second)
	fixedNow := time.Date(2026, 6, 17, 10, 30, 0, 123000000, time.UTC)
	provider.now = func() time.Time { return fixedNow }

	signedURL, err := provider.signDingTalkWebhookURL("https://oapi.dingtalk.com/robot/send?access_token=token", "secret")
	if err != nil {
		t.Fatalf("sign dingtalk webhook url: %v", err)
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	query := parsed.Query()
	timestamp := query.Get("timestamp")
	if timestamp != "1781692200123" {
		t.Fatalf("timestamp = %q", timestamp)
	}
	if query.Get("access_token") != "token" {
		t.Fatalf("access_token should be preserved, query=%s", parsed.RawQuery)
	}
	stringToSign := timestamp + "\nsecret"
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte(stringToSign))
	expectedSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if query.Get("sign") != expectedSign {
		t.Fatalf("sign = %q, want %q", query.Get("sign"), expectedSign)
	}
}
