package repository

import (
	"context"
	"errors"
	"fmt"
	"path"
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

func (r *MessageRepository) ListNotificationRoutes(ctx context.Context, scopePath string) ([]*model.NotificationRouteSetting, error) {
	var rows []*model.NotificationRouteSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	query := r.db.WithContext(ctx)
	if scopePath = NormalizeNotificationScopePath(scopePath); scopePath != "" {
		query = query.Where("scope_path = ?", scopePath)
	}
	err := query.Order("scope_path ASC, channel ASC").Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) ListNotificationRoutesByRoot(ctx context.Context, rootScopePath string) ([]*model.NotificationRouteSetting, error) {
	var rows []*model.NotificationRouteSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	rootScopePath = NormalizeNotificationScopePath(rootScopePath)
	if rootScopePath == "" {
		return []*model.NotificationRouteSetting{}, nil
	}
	err := r.db.WithContext(ctx).
		Where("scope_path = ? OR scope_path LIKE ? ESCAPE '!'", rootScopePath, escapeSQLLikePattern(rootScopePath)+"/%").
		Order("scope_path ASC, channel ASC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) ListEnabledNotificationRoutesByPaths(ctx context.Context, scopePaths []string) ([]*model.NotificationRouteSetting, error) {
	scopePaths = normalizeNotificationScopePaths(scopePaths)
	if len(scopePaths) == 0 {
		return []*model.NotificationRouteSetting{}, nil
	}
	var rows []*model.NotificationRouteSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	err := r.db.WithContext(ctx).
		Where("scope_path IN ? AND enabled = ?", scopePaths, true).
		Order("scope_path DESC, channel ASC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) GetNotificationRoute(ctx context.Context, scopePath, channel string) (*model.NotificationRouteSetting, error) {
	var row model.NotificationRouteSetting
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	err := r.db.WithContext(ctx).
		Where("scope_path = ? AND channel = ?", NormalizeNotificationScopePath(scopePath), normalizeNotificationChannelForRepository(channel)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
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

func (r *MessageRepository) UpsertNotificationRoute(ctx context.Context, setting *model.NotificationRouteSetting) (*model.NotificationRouteSetting, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	if setting == nil {
		return nil, fmt.Errorf("notification route setting is nil")
	}
	setting.ScopePath = NormalizeNotificationScopePath(setting.ScopePath)
	setting.ScopeType = normalizeNotificationScopeType(setting.ScopeType, setting.ScopePath)
	setting.Channel = normalizeNotificationChannelForRepository(setting.Channel)
	setting.DeliveryType = strings.TrimSpace(setting.DeliveryType)
	if setting.DeliveryType == "" {
		setting.DeliveryType = "webhook"
	}
	if setting.ScopePath == "" || setting.Channel == "" {
		return nil, fmt.Errorf("scope_path 和 channel 不能为空")
	}

	existing, err := r.GetNotificationRoute(ctx, setting.ScopePath, setting.Channel)
	if err == nil && existing != nil {
		updates := map[string]interface{}{
			"scope_type":         setting.ScopeType,
			"enabled":            setting.Enabled,
			"delivery_type":      setting.DeliveryType,
			"display_name":       strings.TrimSpace(setting.DisplayName),
			"require_auth":       setting.RequireAuth,
			"webhook_url_cipher": setting.WebhookURLCipher,
			"secret_cipher":      setting.SecretCipher,
			"metadata":           strings.TrimSpace(setting.Metadata),
		}
		if err := r.db.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		return r.GetNotificationRoute(ctx, setting.ScopePath, setting.Channel)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Create(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
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

func (r *MessageRepository) DeleteNotificationRoute(ctx context.Context, scopePath, channel string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	return r.db.WithContext(ctx).
		Where("scope_path = ? AND channel = ?", NormalizeNotificationScopePath(scopePath), normalizeNotificationChannelForRepository(channel)).
		Delete(&model.NotificationRouteSetting{}).Error
}

func (r *MessageRepository) DeleteNotificationChannel(ctx context.Context, ownerUsername, channel string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	return r.db.WithContext(ctx).
		Where("owner_username = ? AND channel = ?", strings.TrimSpace(ownerUsername), normalizeNotificationChannelForRepository(channel)).
		Delete(&model.NotificationChannelSetting{}).Error
}

func (r *MessageRepository) RecordNotificationRouteDeliverySuccess(ctx context.Context, routeID int64, isTest bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	if routeID <= 0 {
		return fmt.Errorf("notification route id is required")
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
		Model(&model.NotificationRouteSetting{}).
		Where("id = ?", routeID).
		Updates(updates).Error
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

func (r *MessageRepository) RecordNotificationRouteDeliveryFailure(ctx context.Context, routeID int64, message string, isTest bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	if routeID <= 0 {
		return fmt.Errorf("notification route id is required")
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
		Model(&model.NotificationRouteSetting{}).
		Where("id = ?", routeID).
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

func NormalizeNotificationScopePath(scopePath string) string {
	scopePath = strings.TrimSpace(scopePath)
	if scopePath == "" {
		return ""
	}
	if !strings.HasPrefix(scopePath, "/") {
		scopePath = "/" + scopePath
	}
	cleaned := path.Clean(scopePath)
	if cleaned == "." || cleaned == "/" {
		return ""
	}
	return cleaned
}

func NotificationRouteCandidatePaths(values ...string) []string {
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	for _, value := range values {
		pathValue := NormalizeNotificationScopePath(value)
		if pathValue == "" {
			continue
		}
		for _, candidate := range notificationScopeAncestors(pathValue) {
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
		}
	}
	return out
}

func notificationScopeAncestors(scopePath string) []string {
	scopePath = NormalizeNotificationScopePath(scopePath)
	if scopePath == "" {
		return nil
	}
	parts := strings.Split(strings.Trim(scopePath, "/"), "/")
	if len(parts) == 0 {
		return nil
	}
	minParts := 1
	if len(parts) >= 2 {
		minParts = 2
	}
	out := make([]string, 0, len(parts)-minParts+1)
	for size := len(parts); size >= minParts; size-- {
		out = append(out, "/"+strings.Join(parts[:size], "/"))
	}
	return out
}

func normalizeNotificationScopePaths(scopePaths []string) []string {
	out := make([]string, 0, len(scopePaths))
	seen := map[string]struct{}{}
	for _, scopePath := range scopePaths {
		scopePath = NormalizeNotificationScopePath(scopePath)
		if scopePath == "" {
			continue
		}
		if _, ok := seen[scopePath]; ok {
			continue
		}
		seen[scopePath] = struct{}{}
		out = append(out, scopePath)
	}
	return out
}

func escapeSQLLikePattern(value string) string {
	value = strings.ReplaceAll(value, `!`, `!!`)
	value = strings.ReplaceAll(value, `%`, `!%`)
	value = strings.ReplaceAll(value, `_`, `!_`)
	return value
}

func normalizeNotificationScopeType(scopeType string, scopePath string) string {
	switch strings.ToLower(strings.TrimSpace(scopeType)) {
	case "workspace", "directory", "function":
		return strings.ToLower(strings.TrimSpace(scopeType))
	}
	scopePath = NormalizeNotificationScopePath(scopePath)
	parts := strings.Split(strings.Trim(scopePath, "/"), "/")
	if len(parts) <= 2 {
		return "workspace"
	}
	last := parts[len(parts)-1]
	if strings.Contains(last, ".") {
		return "function"
	}
	return "directory"
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
