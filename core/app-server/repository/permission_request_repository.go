package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// PermissionRequestRepository 权限申请仓储
type PermissionRequestRepository struct {
	db *gorm.DB
}

// NewPermissionRequestRepository 创建权限申请仓储
func NewPermissionRequestRepository(db *gorm.DB) *PermissionRequestRepository {
	return &PermissionRequestRepository{
		db: db,
	}
}

// GetPermissionRequestByID 根据ID获取权限申请记录
func (r *PermissionRequestRepository) GetPermissionRequestByID(id int64) (*model.PermissionRequest, error) {
	var request model.PermissionRequest
	err := r.db.Where("id = ?", id).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}
