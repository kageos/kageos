package repository

import (
	"context"
	"errors"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SystemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

func (r *SystemSettingRepository) GetAll(ctx context.Context) (map[string]string, error) {
	var rows []model.SystemSetting
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func (r *SystemSettingRepository) Get(ctx context.Context, key string) (string, bool, error) {
	var row model.SystemSetting
	err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return row.Value, true, nil
}

func (r *SystemSettingRepository) UpsertMany(ctx context.Context, values map[string]string, updatedBy string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			row := model.SystemSetting{
				Key:       key,
				Value:     value,
				UpdatedBy: updatedBy,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
