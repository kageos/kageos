package repository

import (
	"context"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

type FunctionSensitiveFieldRepository struct {
	db *gorm.DB
}

func NewFunctionSensitiveFieldRepository(db *gorm.DB) *FunctionSensitiveFieldRepository {
	return &FunctionSensitiveFieldRepository{db: db}
}

func (r *FunctionSensitiveFieldRepository) ReplaceForFunction(ctx context.Context, tenantUser, app, fullCodePath string, fields []*model.FunctionSensitiveField) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_user = ? AND app = ? AND full_code_path = ?", tenantUser, app, fullCodePath).
			Delete(&model.FunctionSensitiveField{}).Error; err != nil {
			return err
		}
		if len(fields) == 0 {
			return nil
		}
		return tx.Create(&fields).Error
	})
}

func (r *FunctionSensitiveFieldRepository) DeleteForFunction(ctx context.Context, tenantUser, app, fullCodePath string) error {
	return r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ? AND full_code_path = ?", tenantUser, app, fullCodePath).
		Delete(&model.FunctionSensitiveField{}).Error
}

func (r *FunctionSensitiveFieldRepository) ListAll(ctx context.Context) ([]*model.FunctionSensitiveField, error) {
	var fields []*model.FunctionSensitiveField
	if err := r.db.WithContext(ctx).Find(&fields).Error; err != nil {
		return nil, err
	}
	return fields, nil
}
