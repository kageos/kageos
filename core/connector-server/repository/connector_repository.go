package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/connector-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConnectorRepository struct {
	db *gorm.DB
}

func NewConnectorRepository(db *gorm.DB) *ConnectorRepository {
	return &ConnectorRepository{db: db}
}

func (r *ConnectorRepository) CreateConnection(ctx context.Context, conn *model.ConnectorConnection) error {
	if conn == nil {
		return fmt.Errorf("connector connection 不能为空")
	}
	return r.db.WithContext(ctx).Create(conn).Error
}

func (r *ConnectorRepository) ListConnections(ctx context.Context, ownerUsername, provider string) ([]*model.ConnectorConnection, error) {
	var rows []*model.ConnectorConnection
	query := r.db.WithContext(ctx).
		Where("owner_username = ? AND status = ?", strings.TrimSpace(ownerUsername), model.ConnectorStatusActive)
	if strings.TrimSpace(provider) != "" {
		query = query.Where("provider = ?", strings.TrimSpace(provider))
	}
	err := query.Order("provider ASC, display_name ASC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *ConnectorRepository) GetOwnedConnection(ctx context.Context, ownerUsername, connectionID string) (*model.ConnectorConnection, error) {
	var row model.ConnectorConnection
	err := r.db.WithContext(ctx).
		Where("owner_username = ? AND connection_id = ? AND status = ?",
			strings.TrimSpace(ownerUsername), strings.TrimSpace(connectionID), model.ConnectorStatusActive).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ConnectorRepository) DeleteOwnedConnection(ctx context.Context, ownerUsername, connectionID string) error {
	ownerUsername = strings.TrimSpace(ownerUsername)
	connectionID = strings.TrimSpace(connectionID)
	if ownerUsername == "" || connectionID == "" {
		return fmt.Errorf("owner_username 和 connection_id 不能为空")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("owner_username = ? AND connection_id = ?", ownerUsername, connectionID).
			Delete(&model.ConnectorDirectoryBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().
			Where("owner_username = ? AND connection_id = ?", ownerUsername, connectionID).
			Delete(&model.ConnectorOAuthToken{}).Error; err != nil {
			return err
		}
		res := tx.Where("owner_username = ? AND connection_id = ?", ownerUsername, connectionID).
			Delete(&model.ConnectorConnection{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *ConnectorRepository) RevokeOwnedConnection(ctx context.Context, ownerUsername, connectionID string) error {
	ownerUsername = strings.TrimSpace(ownerUsername)
	connectionID = strings.TrimSpace(connectionID)
	if ownerUsername == "" || connectionID == "" {
		return fmt.Errorf("owner_username 和 connection_id 不能为空")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("owner_username = ? AND connection_id = ?", ownerUsername, connectionID).
			Delete(&model.ConnectorDirectoryBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().
			Where("owner_username = ? AND connection_id = ?", ownerUsername, connectionID).
			Delete(&model.ConnectorOAuthToken{}).Error; err != nil {
			return err
		}
		res := tx.Model(&model.ConnectorConnection{}).
			Where("owner_username = ? AND connection_id = ? AND status = ?", ownerUsername, connectionID, model.ConnectorStatusActive).
			Updates(map[string]interface{}{
				"status":     model.ConnectorStatusRevoked,
				"updated_at": time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *ConnectorRepository) UpsertDirectoryBinding(ctx context.Context, binding *model.ConnectorDirectoryBinding) error {
	if binding == nil {
		return fmt.Errorf("connector directory binding 不能为空")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "owner_username"},
			{Name: "resource_path"},
			{Name: "provider"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"tenant_user",
			"app",
			"connection_id",
			"updated_by",
			"updated_at",
		}),
	}).Create(binding).Error
}

func (r *ConnectorRepository) DeleteDirectoryBinding(ctx context.Context, ownerUsername, resourcePath, provider string) (int64, error) {
	res := r.db.WithContext(ctx).Unscoped().
		Where("owner_username = ? AND resource_path = ? AND provider = ?",
			strings.TrimSpace(ownerUsername), strings.TrimSpace(resourcePath), strings.TrimSpace(provider)).
		Delete(&model.ConnectorDirectoryBinding{})
	return res.RowsAffected, res.Error
}

func (r *ConnectorRepository) ListDirectoryBindings(ctx context.Context, ownerUsername, resourcePath, provider string) ([]*model.ConnectorDirectoryBinding, error) {
	var rows []*model.ConnectorDirectoryBinding
	query := r.db.WithContext(ctx).Where("owner_username = ?", strings.TrimSpace(ownerUsername))
	if strings.TrimSpace(resourcePath) != "" {
		query = query.Where("resource_path = ?", strings.TrimSpace(resourcePath))
	}
	if strings.TrimSpace(provider) != "" {
		query = query.Where("provider = ?", strings.TrimSpace(provider))
	}
	err := query.Order("resource_path ASC, provider ASC").Find(&rows).Error
	return rows, err
}

func (r *ConnectorRepository) FindDirectoryBinding(ctx context.Context, ownerUsername, resourcePath, provider string) (*model.ConnectorDirectoryBinding, error) {
	var row model.ConnectorDirectoryBinding
	err := r.db.WithContext(ctx).
		Where("owner_username = ? AND resource_path = ? AND provider = ?",
			strings.TrimSpace(ownerUsername), strings.TrimSpace(resourcePath), strings.TrimSpace(provider)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ConnectorRepository) CreateOAuthState(ctx context.Context, state *model.ConnectorOAuthState) error {
	if state == nil {
		return fmt.Errorf("oauth state 不能为空")
	}
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *ConnectorRepository) GetPendingOAuthState(ctx context.Context, state string) (*model.ConnectorOAuthState, error) {
	var row model.ConnectorOAuthState
	err := r.db.WithContext(ctx).
		Where("state = ? AND status = ? AND expires_at > ?",
			strings.TrimSpace(state), model.OAuthStateStatusPending, time.Now()).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ConnectorRepository) MarkOAuthStateUsed(ctx context.Context, state string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&model.ConnectorOAuthState{}).
		Where("state = ? AND status = ?", strings.TrimSpace(state), model.OAuthStateStatusPending).
		Updates(map[string]interface{}{
			"status":     model.OAuthStateStatusUsed,
			"used_at":    &now,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ConnectorRepository) UpsertOAuthToken(ctx context.Context, token *model.ConnectorOAuthToken) error {
	if token == nil {
		return fmt.Errorf("oauth token 不能为空")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_username",
			"provider",
			"access_token_cipher",
			"refresh_token_cipher",
			"token_type",
			"scopes",
			"expiry",
			"last_refresh_at",
			"raw_response",
			"updated_by",
			"updated_at",
		}),
	}).Create(token).Error
}

func (r *ConnectorRepository) GetOwnedOAuthToken(ctx context.Context, ownerUsername, connectionID string) (*model.ConnectorOAuthToken, error) {
	var row model.ConnectorOAuthToken
	err := r.db.WithContext(ctx).
		Where("owner_username = ? AND connection_id = ?", strings.TrimSpace(ownerUsername), strings.TrimSpace(connectionID)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ConnectorRepository) UpsertOAuthProviderSetting(ctx context.Context, setting *model.ConnectorOAuthProviderSetting) error {
	if setting == nil {
		return fmt.Errorf("oauth provider setting 不能为空")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"client_id",
			"client_secret_cipher",
			"auth_url",
			"token_url",
			"user_info_url",
			"scopes",
			"use_pkce",
			"token_request_mode",
			"client_id_param",
			"client_secret_param",
			"grant_type_param",
			"code_param",
			"refresh_token_param",
			"redirect_uri_param",
			"extra_auth_params",
			"extra_token_params",
			"external_id_field",
			"display_name_field",
			"enabled",
			"updated_by",
			"updated_at",
		}),
	}).Create(setting).Error
}

func (r *ConnectorRepository) CreateOAuthProviderSettingIfNotExists(ctx context.Context, setting *model.ConnectorOAuthProviderSetting) error {
	if setting == nil {
		return fmt.Errorf("oauth provider setting 不能为空")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(setting).Error
}

func (r *ConnectorRepository) ListOAuthProviderSettings(ctx context.Context) ([]*model.ConnectorOAuthProviderSetting, error) {
	var rows []*model.ConnectorOAuthProviderSetting
	err := r.db.WithContext(ctx).Order("code ASC").Find(&rows).Error
	return rows, err
}

func (r *ConnectorRepository) GetOAuthProviderSetting(ctx context.Context, code string) (*model.ConnectorOAuthProviderSetting, error) {
	var row model.ConnectorOAuthProviderSetting
	err := r.db.WithContext(ctx).Where("code = ?", normalizeCode(code)).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ConnectorRepository) DeleteOAuthProviderSetting(ctx context.Context, code string) (int64, error) {
	res := r.db.WithContext(ctx).Unscoped().
		Where("code = ?", normalizeCode(code)).
		Delete(&model.ConnectorOAuthProviderSetting{})
	return res.RowsAffected, res.Error
}

func normalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
