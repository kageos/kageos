package repository

import (
	"context"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// PermissionRequestRepository 权限申请仓储
type PermissionRequestRepository struct {
	db *gorm.DB
}

// NewPermissionRequestRepository 创建权限申请仓储
func NewPermissionRequestRepository(db *gorm.DB) *PermissionRequestRepository {
	return &PermissionRequestRepository{db: db}
}

// SaveRequest 保存申请记录
func (r *PermissionRequestRepository) SaveRequest(ctx context.Context, request *model.PermissionRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// GetRequestByID 根据ID查询申请记录
func (r *PermissionRequestRepository) GetRequestByID(ctx context.Context, id int64) (*model.PermissionRequest, error) {
	var request model.PermissionRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// UpdateRequestStatus 更新申请状态
func (r *PermissionRequestRepository) UpdateRequestStatus(
	ctx context.Context,
	id int64,
	status string,
	approvedBy string,
	rejectedBy string,
	rejectReason string,
) error {
	updates := map[string]interface{}{
		"status": status,
	}
	
	if status == model.PermissionRequestStatusApproved {
		now := time.Now()
		updates["approved_at"] = &now
		updates["approved_by"] = approvedBy
	} else if status == model.PermissionRequestStatusRejected {
		now := time.Now()
		updates["rejected_at"] = &now
		updates["rejected_by"] = rejectedBy
		updates["reject_reason"] = rejectReason
	}

	return r.db.WithContext(ctx).Model(&model.PermissionRequest{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetPendingRequestsByResource 查询资源的待审批申请
func (r *PermissionRequestRepository) GetPendingRequestsByResource(
	ctx context.Context,
	resourcePath string,
) ([]*model.PermissionRequest, error) {
	var requests []*model.PermissionRequest
	err := r.db.WithContext(ctx).
		Where("resource_path = ? AND status = ?", resourcePath, model.PermissionRequestStatusPending).
		Find(&requests).Error
	if err != nil {
		return nil, err
	}
	return requests, nil
}

// GetPendingRequestsByApprover 查询审批人的待审批申请
func (r *PermissionRequestRepository) GetPendingRequestsByApprover(
	ctx context.Context,
	approverUsername string,
	appID int64,
) ([]*model.PermissionRequest, error) {
	// 这里需要根据审批策略查询，暂时先返回空
	// 后续实现审批策略解析后，再实现此方法
	return []*model.PermissionRequest{}, nil
}

// UpdateRoleAssignmentID 更新申请记录的 role_assignment_id
func (r *PermissionRequestRepository) UpdateRoleAssignmentID(ctx context.Context, requestID int64, roleAssignmentID int64) error {
	return r.db.WithContext(ctx).
		Model(&model.PermissionRequest{}).
		Where("id = ?", requestID).
		Update("role_assignment_id", roleAssignmentID).Error
}

