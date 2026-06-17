package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/message-server/model"
	"gorm.io/gorm"
)

func (r *MessageRepository) ListNotificationChannels(ctx context.Context, ownerUsername string) ([]*model.NotificationChannelSetting, error) {
	var rows []*model.NotificationChannelSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	err := r.db.WithContext(ctx).
		Where("owner_username = ?", strings.TrimSpace(ownerUsername)).
		Order("channel ASC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) ListEnabledNotificationChannels(ctx context.Context, usernames []string) ([]*model.NotificationChannelSetting, error) {
	usernames = normalizeUsernames(usernames)
	if len(usernames) == 0 {
		return []*model.NotificationChannelSetting{}, nil
	}
	var rows []*model.NotificationChannelSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	err := r.db.WithContext(ctx).
		Where("owner_username IN ? AND enabled = ?", usernames, true).
		Order("owner_username ASC, channel ASC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) GetNotificationChannel(ctx context.Context, ownerUsername, channel string) (*model.NotificationChannelSetting, error) {
	var row model.NotificationChannelSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	err := r.db.WithContext(ctx).
		Where("owner_username = ? AND channel = ?", strings.TrimSpace(ownerUsername), normalizeNotificationChannelForRepository(channel)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *MessageRepository) UpsertNotificationChannel(ctx context.Context, setting *model.NotificationChannelSetting) (*model.NotificationChannelSetting, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	if setting == nil {
		return nil, fmt.Errorf("notification channel setting is nil")
	}
	setting.OwnerUsername = strings.TrimSpace(setting.OwnerUsername)
	setting.Channel = normalizeNotificationChannelForRepository(setting.Channel)
	setting.DeliveryType = strings.TrimSpace(setting.DeliveryType)
	if setting.DeliveryType == "" {
		setting.DeliveryType = "webhook"
	}
	if setting.OwnerUsername == "" || setting.Channel == "" {
		return nil, fmt.Errorf("owner_username 和 channel 不能为空")
	}

	existing, err := r.GetNotificationChannel(ctx, setting.OwnerUsername, setting.Channel)
	if err == nil && existing != nil {
		updates := map[string]interface{}{
			"enabled":            setting.Enabled,
			"delivery_type":      setting.DeliveryType,
			"display_name":       strings.TrimSpace(setting.DisplayName),
			"webhook_url_cipher": setting.WebhookURLCipher,
			"secret_cipher":      setting.SecretCipher,
			"metadata":           strings.TrimSpace(setting.Metadata),
		}
		if err := r.db.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		return r.GetNotificationChannel(ctx, setting.OwnerUsername, setting.Channel)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Create(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
}

func (r *MessageRepository) DeleteNotificationChannel(ctx context.Context, ownerUsername, channel string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	return r.db.WithContext(ctx).
		Where("owner_username = ? AND channel = ?", strings.TrimSpace(ownerUsername), normalizeNotificationChannelForRepository(channel)).
		Delete(&model.NotificationChannelSetting{}).Error
}

func (r *MessageRepository) RecordNotificationChannelDeliverySuccess(ctx context.Context, ownerUsername, channel string, isTest bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"last_success_at": now,
		"last_error":      "",
		"fail_count":      0,
	}
	if isTest {
		updates["last_test_at"] = now
	}
	return r.db.WithContext(ctx).
		Model(&model.NotificationChannelSetting{}).
		Where("owner_username = ? AND channel = ?", strings.TrimSpace(ownerUsername), normalizeNotificationChannelForRepository(channel)).
		Updates(updates).Error
}

func (r *MessageRepository) RecordNotificationChannelDeliveryFailure(ctx context.Context, ownerUsername, channel, message string, isTest bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"last_failed_at": now,
		"last_error":     truncateNotificationDeliveryError(message, 2000),
		"fail_count":     gorm.Expr("COALESCE(fail_count, 0) + ?", 1),
	}
	if isTest {
		updates["last_test_at"] = now
	}
	return r.db.WithContext(ctx).
		Model(&model.NotificationChannelSetting{}).
		Where("owner_username = ? AND channel = ?", strings.TrimSpace(ownerUsername), normalizeNotificationChannelForRepository(channel)).
		Updates(updates).Error
}

func normalizeNotificationChannelForRepository(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func truncateNotificationDeliveryError(message string, maxRunes int) string {
	message = strings.TrimSpace(message)
	if maxRunes <= 0 {
		return message
	}
	runes := []rune(message)
	if len(runes) <= maxRunes {
		return message
	}
	return string(runes[:maxRunes]) + "..."
}
