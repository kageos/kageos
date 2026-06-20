package repository

import (
	"errors"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthLoginProviderRepository struct {
	db *gorm.DB
}

func NewAuthLoginProviderRepository(db *gorm.DB) *AuthLoginProviderRepository {
	return &AuthLoginProviderRepository{db: db}
}

func (r *AuthLoginProviderRepository) List() ([]*model.AuthLoginProvider, error) {
	var providers []*model.AuthLoginProvider
	err := r.db.Order("sort_order ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (r *AuthLoginProviderRepository) GetByCode(code string) (*model.AuthLoginProvider, error) {
	var provider model.AuthLoginProvider
	err := r.db.Where("code = ?", code).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (r *AuthLoginProviderRepository) UpsertSeed(provider *model.AuthLoginProvider) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"description",
			"action",
			"config_schema_json",
			"callback_path",
			"docs_url",
			"sort_order",
			"updated_at",
		}),
	}).Create(provider).Error
}

func (r *AuthLoginProviderRepository) UpdateConfig(provider *model.AuthLoginProvider) error {
	return r.db.Model(&model.AuthLoginProvider{}).
		Where("code = ?", provider.Code).
		Updates(map[string]interface{}{
			"enabled":            provider.Enabled,
			"configured":         provider.Configured,
			"status":             provider.Status,
			"config_values_json": provider.ConfigValuesJSON,
			"updated_by":         provider.UpdatedBy,
		}).Error
}

func (r *AuthLoginProviderRepository) UpdateEnabled(code string, enabled bool, status string, updatedBy string) error {
	return r.db.Model(&model.AuthLoginProvider{}).
		Where("code = ?", code).
		Updates(map[string]interface{}{
			"enabled":    enabled,
			"status":     status,
			"updated_by": updatedBy,
		}).Error
}
