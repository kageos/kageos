package server

import (
	"testing"
	"time"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/core/message-server/service"
)

func TestValidateNotificationWebhookURLForServer(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		raw     string
		wantErr bool
	}{
		{
			name:    "feishu",
			channel: service.NotificationChannelFeishu,
			raw:     "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
		},
		{
			name:    "larksuite",
			channel: service.NotificationChannelFeishu,
			raw:     "https://open.larksuite.com/open-apis/bot/v2/hook/test-token",
		},
		{
			name:    "wecom",
			channel: service.NotificationChannelWeCom,
			raw:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token",
		},
		{
			name:    "dingtalk",
			channel: service.NotificationChannelDingTalk,
			raw:     "https://oapi.dingtalk.com/robot/send?access_token=test-token",
		},
		{
			name:    "wrong host",
			channel: service.NotificationChannelWeCom,
			raw:     "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
			wantErr: true,
		},
		{
			name:    "http rejected",
			channel: service.NotificationChannelDingTalk,
			raw:     "http://oapi.dingtalk.com/robot/send?access_token=test-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationWebhookURLForServer(tt.channel, tt.raw)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNotificationChannelToInfoIncludesDeliveryStatus(t *testing.T) {
	now := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)
	row := &msgmodel.NotificationChannelSetting{
		Channel:          service.NotificationChannelDingTalk,
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "钉钉群",
		WebhookURLCipher: "cipher-url",
		SecretCipher:     "cipher-secret",
		Metadata:         `{"keyword":"Kageos"}`,
		LastSuccessAt:    &now,
		LastFailedAt:     &now,
		LastTestAt:       &now,
		LastError:        "webhook returned 400",
		FailCount:        2,
	}

	info := notificationChannelToInfo(row)
	if info.Channel != service.NotificationChannelDingTalk || !info.Enabled || !info.HasWebhookURL || !info.HasSecret {
		t.Fatalf("unexpected basic info: %#v", info)
	}
	if info.Metadata["keyword"] != "Kageos" {
		t.Fatalf("metadata = %#v", info.Metadata)
	}
	if info.LastSuccessAt == nil || info.LastFailedAt == nil || info.LastTestAt == nil {
		t.Fatalf("missing delivery timestamps: %#v", info)
	}
	if info.LastError != "webhook returned 400" || info.FailCount != 2 {
		t.Fatalf("unexpected delivery status: %#v", info)
	}
}

func TestNotificationRouteToInfoIncludesRemarkAndDeliveryStatus(t *testing.T) {
	now := time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC)
	row := &msgmodel.NotificationRouteSetting{
		ID:               42,
		ScopePath:        "/alice/sales/orders",
		ScopeType:        "directory",
		Channel:          service.NotificationChannelFeishu,
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "订单群",
		Remark:           "飞书订单通知群",
		WebhookURLCipher: "cipher-url",
		SecretCipher:     "cipher-secret",
		Metadata:         `{"tenant":"demo"}`,
		UpdatedAt:        now,
		LastSuccessAt:    &now,
		LastFailedAt:     &now,
		LastTestAt:       &now,
		LastError:        "webhook returned 400",
		FailCount:        2,
	}

	info := notificationRouteToInfo(row)
	if info.ID != 42 || info.ScopePath != "/alice/sales/orders" || info.Channel != service.NotificationChannelFeishu {
		t.Fatalf("unexpected basic info: %#v", info)
	}
	if info.DisplayName != "订单群" || info.Remark != "飞书订单通知群" {
		t.Fatalf("unexpected display fields: %#v", info)
	}
	if !info.HasWebhookURL || !info.HasSecret {
		t.Fatalf("unexpected route flags: %#v", info)
	}
	if info.Metadata["tenant"] != "demo" || info.LastError != "webhook returned 400" || info.FailCount != 2 {
		t.Fatalf("unexpected delivery fields: %#v", info)
	}
}
