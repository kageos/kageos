package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type WebhookChannelProvider struct {
	channel  string
	renderer CardRenderer
	client   *http.Client
	now      func() time.Time
}

func NewFeishuWebhookChannelProvider(timeout time.Duration) *WebhookChannelProvider {
	return NewWebhookChannelProvider(NotificationChannelFeishu, FeishuCardRenderer{}, timeout)
}

func NewWeComWebhookChannelProvider(timeout time.Duration) *WebhookChannelProvider {
	return NewWebhookChannelProvider(NotificationChannelWeCom, WeComTemplateCardRenderer{}, timeout)
}

func NewDingTalkWebhookChannelProvider(timeout time.Duration) *WebhookChannelProvider {
	return NewWebhookChannelProvider(NotificationChannelDingTalk, DingTalkActionCardRenderer{}, timeout)
}

func NewWebhookChannelProvider(channel string, renderer CardRenderer, timeout time.Duration) *WebhookChannelProvider {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookChannelProvider{
		channel:  normalizeNotificationChannel(channel),
		renderer: renderer,
		client:   &http.Client{Timeout: timeout},
		now:      time.Now,
	}
}

func (p *WebhookChannelProvider) Channel() string {
	if p == nil {
		return ""
	}
	return p.channel
}

func (p *WebhookChannelProvider) Deliver(ctx context.Context, target NotificationTarget, card NotificationCard) error {
	if p == nil || p.renderer == nil {
		return fmt.Errorf("webhook channel provider is not initialized")
	}
	webhookURL := strings.TrimSpace(target.WebhookURL)
	if webhookURL == "" {
		return fmt.Errorf("%s webhook url is empty", p.Channel())
	}
	payload, err := p.renderer.Render(card)
	if err != nil {
		return err
	}
	if p.channel == NotificationChannelFeishu && strings.TrimSpace(target.Secret) != "" {
		p.applyFeishuSignature(payload, target.Secret)
	}
	if p.channel == NotificationChannelDingTalk && strings.TrimSpace(target.Secret) != "" {
		signedURL, err := p.signDingTalkWebhookURL(webhookURL, target.Secret)
		if err != nil {
			return err
		}
		webhookURL = signedURL
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s webhook payload: %w", p.Channel(), err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s webhook request: %w", p.Channel(), err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post %s webhook: %w", p.Channel(), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s webhook returned %s: %s", p.Channel(), resp.Status, strings.TrimSpace(string(raw)))
	}
	if err := validateWebhookResponseBody(p.Channel(), raw); err != nil {
		return err
	}
	return nil
}

func validateWebhookResponseBody(channel string, raw []byte) error {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil
	}
	for _, key := range []string{"errcode", "code", "StatusCode", "status_code"} {
		if value, ok := body[key]; ok && !isWebhookSuccessCode(value) {
			message := firstNonEmptyString(interfaceString(body["errmsg"]), interfaceString(body["msg"]), interfaceString(body["StatusMessage"]), strings.TrimSpace(string(raw)))
			return fmt.Errorf("%s webhook returned %s=%v: %s", channel, key, value, message)
		}
	}
	if value, ok := body["success"].(bool); ok && !value {
		return fmt.Errorf("%s webhook returned success=false: %s", channel, strings.TrimSpace(string(raw)))
	}
	return nil
}

func isWebhookSuccessCode(value interface{}) bool {
	switch v := value.(type) {
	case float64:
		return v == 0
	case string:
		v = strings.TrimSpace(v)
		return v == "" || v == "0"
	case nil:
		return true
	default:
		return fmt.Sprint(v) == "0"
	}
}

func interfaceString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (p *WebhookChannelProvider) applyFeishuSignature(payload map[string]interface{}, secret string) {
	now := time.Now
	if p != nil && p.now != nil {
		now = p.now
	}
	timestamp := strconv.FormatInt(now().Unix(), 10)
	stringToSign := timestamp + "\n" + strings.TrimSpace(secret)
	mac := hmac.New(sha256.New, []byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	payload["timestamp"] = timestamp
	payload["sign"] = sign
}

func (p *WebhookChannelProvider) signDingTalkWebhookURL(webhookURL string, secret string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(webhookURL))
	if err != nil {
		return "", fmt.Errorf("parse dingtalk webhook url: %w", err)
	}
	now := time.Now
	if p != nil && p.now != nil {
		now = p.now
	}
	trimmedSecret := strings.TrimSpace(secret)
	timestamp := strconv.FormatInt(now().UnixMilli(), 10)
	stringToSign := timestamp + "\n" + trimmedSecret
	mac := hmac.New(sha256.New, []byte(trimmedSecret))
	_, _ = mac.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", sign)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
