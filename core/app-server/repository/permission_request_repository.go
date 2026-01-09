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

// GetPermissionRequestsWithPage 获取权限申请列表（支持筛选和分页）
func (r *PermissionRequestRepository) GetPermissionRequestsWithPage(
	appID int64,
	status string,
	applicant string,
	resourcePath string,
	page int,
	pageSize int,
) ([]*model.PermissionRequest, int64, error) {
	var requests []*model.PermissionRequest
	var totalCount int64

	// 构建查询条件
	query := r.db.Model(&model.PermissionRequest{})

	// 筛选条件
	if appID > 0 {
		query = query.Where("app_id = ?", appID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if applicant != "" {
		query = query.Where("applicant_username = ?", applicant)
	}
	if resourcePath != "" {
		query = query.Where("resource_path = ?", resourcePath)
	}

	// 获取总数
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据，按创建时间倒序
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error
	if err != nil {
		return nil, 0, err
	}

	return requests, totalCount, nil
}
