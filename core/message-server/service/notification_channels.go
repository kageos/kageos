package service

import (
	"fmt"
	"time"
)

var supportedNotificationChannels = map[string]struct{}{
	NotificationChannelFeishu:   {},
	NotificationChannelWeCom:    {},
	NotificationChannelDingTalk: {},
}

func IsSupportedNotificationChannel(channel string) bool {
	_, ok := supportedNotificationChannels[normalizeNotificationChannel(channel)]
	return ok
}

func NewDefaultNotificationChannelProviders(timeout time.Duration) []ChannelProvider {
	return []ChannelProvider{
		NewFeishuWebhookChannelProvider(timeout),
		NewWeComWebhookChannelProvider(timeout),
		NewDingTalkWebhookChannelProvider(timeout),
	}
}

func NewNotificationChannelProvider(channel string, timeout time.Duration) (ChannelProvider, error) {
	switch normalizeNotificationChannel(channel) {
	case NotificationChannelFeishu:
		return NewFeishuWebhookChannelProvider(timeout), nil
	case NotificationChannelWeCom:
		return NewWeComWebhookChannelProvider(timeout), nil
	case NotificationChannelDingTalk:
		return NewDingTalkWebhookChannelProvider(timeout), nil
	default:
		return nil, fmt.Errorf("不支持的通知渠道: %s", channel)
	}
}
