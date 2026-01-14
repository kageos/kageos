package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// RolePermissionRepository 角色权限仓储
type RolePermissionRepository struct {
	db *gorm.DB
}

// NewRolePermissionRepository 创建角色权限仓储
func NewRolePermissionRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

// CreateRolePermission 创建角色权限
func (r *RolePermissionRepository) CreateRolePermission(ctx context.Context, rolePerm *model.RolePermission) error {
	return r.db.WithContext(ctx).Create(rolePerm).Error
}

// GetPermissionsByRoleIDAndResourceType 根据角色ID和资源类型获取权限列表
func (r *RolePermissionRepository) GetPermissionsByRoleIDAndResourceType(ctx context.Context, roleID int64, resourceType string) ([]*model.RolePermission, error) {
	var perms []*model.RolePermission
	query := r.db.WithContext(ctx).Where("role_id = ?", roleID)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	err := query.Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// GetPermissionsByRoleID 根据角色ID获取权限列表
func (r *RolePermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID int64) ([]*model.RolePermission, error) {
	var perms []*model.RolePermission
	err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// GetPermissionsByRoleIDs 根据角色ID列表获取权限列表（批量查询）
func (r *RolePermissionRepository) GetPermissionsByRoleIDs(ctx context.Context, roleIDs []int64) ([]*model.RolePermission, error) {
	if len(roleIDs) == 0 {
		return []*model.RolePermission{}, nil
	}
	
	var perms []*model.RolePermission
	err := r.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// GetAllRolePermissions 获取所有角色权限（用于缓存加载）
func (r *RolePermissionRepository) GetAllRolePermissions(ctx context.Context) ([]*model.RolePermission, error) {
	var perms []*model.RolePermission
	err := r.db.WithContext(ctx).Find(&perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// DeleteByRoleID 删除角色的所有权限
func (r *RolePermissionRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error
}

// DeleteRolePermission 删除单个角色权限
func (r *RolePermissionRepository) DeleteRolePermission(ctx context.Context, roleID int64, action string) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND action = ?", roleID, action).Delete(&model.RolePermission{}).Error
}
