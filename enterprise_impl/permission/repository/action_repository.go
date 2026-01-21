package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// ActionRepository 权限点仓储
type ActionRepository struct {
	db *gorm.DB
}

// NewActionRepository 创建权限点仓储
func NewActionRepository(db *gorm.DB) *ActionRepository {
	return &ActionRepository{db: db}
}

// CreateAction 创建权限点
func (r *ActionRepository) CreateAction(ctx context.Context, action *model.Action) error {
	return r.db.WithContext(ctx).Create(action).Error
}

// GetActionByCode 根据编码获取权限点
func (r *ActionRepository) GetActionByCode(ctx context.Context, code string) (*model.Action, error) {
	var action model.Action
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&action).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

// GetActionsByResourceType 根据资源类型获取权限点列表
func (r *ActionRepository) GetActionsByResourceType(ctx context.Context, resourceType string) ([]*model.Action, error) {
	var actions []*model.Action
	err := r.db.WithContext(ctx).Where("resource_type = ?", resourceType).Order("action_type ASC").Find(&actions).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}

// GetAllActions 获取所有权限点
func (r *ActionRepository) GetAllActions(ctx context.Context) ([]*model.Action, error) {
	var actions []*model.Action
	err := r.db.WithContext(ctx).Order("resource_type ASC, action_type ASC").Find(&actions).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}

// CountActions 统计权限点数量
func (r *ActionRepository) CountActions(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Action{}).Count(&count).Error
	return count, err
}
