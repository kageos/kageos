package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/dto"
)

func normalizeNotificationChannelParam(channel string) (string, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if service.IsSupportedNotificationChannel(channel) {
		return channel, nil
	}
	return "", fmt.Errorf("不支持的通知渠道: %s", channel)
}

func notificationTestProvider(channel string, timeout time.Duration) (service.ChannelProvider, error) {
	return service.NewNotificationChannelProvider(channel, timeout)
}

func validateNotificationWebhookURLForServer(channel string, raw string) error {
	return service.ValidateNotificationWebhookURL(channel, raw)
}

func notificationChannelToInfo(row *msgmodel.NotificationChannelSetting) dto.MessageNotificationChannelInfo {
	if row == nil {
		return dto.MessageNotificationChannelInfo{}
	}
	return dto.MessageNotificationChannelInfo{
		Channel:       row.Channel,
		Enabled:       row.Enabled,
		DeliveryType:  firstNonEmptyStringForServer(row.DeliveryType, "webhook"),
		DisplayName:   row.DisplayName,
		HasWebhookURL: strings.TrimSpace(row.WebhookURLCipher) != "",
		HasSecret:     strings.TrimSpace(row.SecretCipher) != "",
		Metadata:      parseStringMetadataForServer(row.Metadata),
		UpdatedAt:     row.UpdatedAt,
		LastSuccessAt: row.LastSuccessAt,
		LastFailedAt:  row.LastFailedAt,
		LastTestAt:    row.LastTestAt,
		LastError:     row.LastError,
		FailCount:     row.FailCount,
	}
}

func notificationRouteToInfo(row *msgmodel.NotificationRouteSetting) dto.MessageNotificationRouteInfo {
	if row == nil {
		return dto.MessageNotificationRouteInfo{}
	}
	return dto.MessageNotificationRouteInfo{
		ID:            row.ID,
		ScopePath:     row.ScopePath,
		ScopeType:     row.ScopeType,
		Channel:       row.Channel,
		Enabled:       row.Enabled,
		DeliveryType:  firstNonEmptyStringForServer(row.DeliveryType, "webhook"),
		DisplayName:   row.DisplayName,
		Remark:        row.Remark,
		HasWebhookURL: strings.TrimSpace(row.WebhookURLCipher) != "",
		HasSecret:     strings.TrimSpace(row.SecretCipher) != "",
		Metadata:      parseStringMetadataForServer(row.Metadata),
		UpdatedAt:     row.UpdatedAt,
		LastSuccessAt: row.LastSuccessAt,
		LastFailedAt:  row.LastFailedAt,
		LastTestAt:    row.LastTestAt,
		LastError:     row.LastError,
		FailCount:     row.FailCount,
	}
}

func marshalStringMetadataForServer(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	clean := make(map[string]string, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		clean[key] = strings.TrimSpace(value)
	}
	if len(clean) == 0 {
		return ""
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return string(raw)
}

func parseStringMetadataForServer(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmptyStringForServer(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
