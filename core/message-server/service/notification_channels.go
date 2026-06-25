package service

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ChannelProviderFactory func(timeout time.Duration) ChannelProvider

type WebhookURLValidator func(raw string) error

// NotificationChannelDefinition keeps channel creation and validation together.
type NotificationChannelDefinition struct {
	Channel            string
	ProviderFactory    ChannelProviderFactory
	ValidateWebhookURL WebhookURLValidator
}

var notificationChannelRegistry = struct {
	sync.RWMutex
	order    []string
	channels map[string]NotificationChannelDefinition
}{
	channels: make(map[string]NotificationChannelDefinition),
}

// RegisterNotificationChannel registers a notification channel end to end.
func RegisterNotificationChannel(definition NotificationChannelDefinition) {
	definition.Channel = normalizeNotificationChannel(definition.Channel)
	if definition.Channel == "" || definition.ProviderFactory == nil {
		panic("notification channel requires channel and provider factory")
	}
	notificationChannelRegistry.Lock()
	defer notificationChannelRegistry.Unlock()
	if _, exists := notificationChannelRegistry.channels[definition.Channel]; exists {
		panic(fmt.Sprintf("notification channel %s already registered", definition.Channel))
	}
	notificationChannelRegistry.order = append(notificationChannelRegistry.order, definition.Channel)
	notificationChannelRegistry.channels[definition.Channel] = definition
}

func LookupNotificationChannel(channel string) (NotificationChannelDefinition, bool) {
	channel = normalizeNotificationChannel(channel)
	notificationChannelRegistry.RLock()
	defer notificationChannelRegistry.RUnlock()
	definition, ok := notificationChannelRegistry.channels[channel]
	return definition, ok
}

func RegisteredNotificationChannels() []string {
	notificationChannelRegistry.RLock()
	defer notificationChannelRegistry.RUnlock()
	out := make([]string, len(notificationChannelRegistry.order))
	copy(out, notificationChannelRegistry.order)
	return out
}

func IsSupportedNotificationChannel(channel string) bool {
	channel = normalizeNotificationChannel(channel)
	notificationChannelRegistry.RLock()
	defer notificationChannelRegistry.RUnlock()
	_, ok := notificationChannelRegistry.channels[channel]
	return ok
}

func NewDefaultNotificationChannelProviders(timeout time.Duration) []ChannelProvider {
	notificationChannelRegistry.RLock()
	factories := make([]ChannelProviderFactory, 0, len(notificationChannelRegistry.order))
	for _, channel := range notificationChannelRegistry.order {
		if definition := notificationChannelRegistry.channels[channel]; definition.ProviderFactory != nil {
			factories = append(factories, definition.ProviderFactory)
		}
	}
	notificationChannelRegistry.RUnlock()

	providers := make([]ChannelProvider, 0, len(factories))
	for _, factory := range factories {
		if provider := factory(timeout); provider != nil {
			providers = append(providers, provider)
		}
	}
	return providers
}

func NewNotificationChannelProvider(channel string, timeout time.Duration) (ChannelProvider, error) {
	channel = normalizeNotificationChannel(channel)
	notificationChannelRegistry.RLock()
	definition := notificationChannelRegistry.channels[channel]
	notificationChannelRegistry.RUnlock()
	if definition.ProviderFactory == nil {
		return nil, fmt.Errorf("不支持的通知渠道: %s", channel)
	}
	provider := definition.ProviderFactory(timeout)
	if provider == nil {
		return nil, fmt.Errorf("通知渠道 %s 未正确初始化", channel)
	}
	return provider, nil
}

func ValidateNotificationWebhookURL(channel string, raw string) error {
	channel = normalizeNotificationChannel(channel)
	definition, ok := LookupNotificationChannel(channel)
	if !ok {
		return fmt.Errorf("不支持的通知渠道: %s", channel)
	}
	if definition.ValidateWebhookURL != nil {
		return definition.ValidateWebhookURL(raw)
	}
	_, err := parseHTTPSWebhookURL(channel, raw)
	return err
}

func init() {
	RegisterNotificationChannel(NotificationChannelDefinition{
		Channel: NotificationChannelFeishu,
		ProviderFactory: func(timeout time.Duration) ChannelProvider {
			return NewFeishuWebhookChannelProvider(timeout)
		},
		ValidateWebhookURL: validateFeishuWebhookURL,
	})
	RegisterNotificationChannel(NotificationChannelDefinition{
		Channel: NotificationChannelWeCom,
		ProviderFactory: func(timeout time.Duration) ChannelProvider {
			return NewWeComWebhookChannelProvider(timeout)
		},
		ValidateWebhookURL: validateWeComWebhookURL,
	})
	RegisterNotificationChannel(NotificationChannelDefinition{
		Channel: NotificationChannelDingTalk,
		ProviderFactory: func(timeout time.Duration) ChannelProvider {
			return NewDingTalkWebhookChannelProvider(timeout)
		},
		ValidateWebhookURL: validateDingTalkWebhookURL,
	})
}

func validateFeishuWebhookURL(raw string) error {
	return validateWebhookURL(NotificationChannelFeishu, raw, func(host, path string) bool {
		return (host == "open.feishu.cn" || host == "open.larksuite.com") && strings.HasPrefix(path, "/open-apis/bot/")
	})
}

func validateWeComWebhookURL(raw string) error {
	return validateWebhookURL(NotificationChannelWeCom, raw, func(host, path string) bool {
		return host == "qyapi.weixin.qq.com" && path == "/cgi-bin/webhook/send"
	})
}

func validateDingTalkWebhookURL(raw string) error {
	return validateWebhookURL(NotificationChannelDingTalk, raw, func(host, path string) bool {
		return host == "oapi.dingtalk.com" && path == "/robot/send"
	})
}

func validateWebhookURL(channel, raw string, match func(host, path string) bool) error {
	parsed, err := parseHTTPSWebhookURL(channel, raw)
	if err != nil {
		return err
	}
	if match(strings.ToLower(parsed.Hostname()), parsed.EscapedPath()) {
		return nil
	}
	return fmt.Errorf("%s Webhook 地址与当前渠道不匹配", channel)
}

func parseHTTPSWebhookURL(channel, raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("%s Webhook 地址格式不正确", channel)
	}
	return parsed, nil
}
