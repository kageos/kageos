package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// RoleRepository 角色仓储
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色仓储
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// CreateRole 创建角色
func (r *RoleRepository) CreateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// GetRoleByID 根据ID获取角色
func (r *RoleRepository) GetRoleByID(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRoleByCode 根据编码获取角色（需要同时匹配 code 和 resourceType）
func (r *RoleRepository) GetRoleByCode(ctx context.Context, code string, resourceType string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("code = ? AND resource_type = ?", code, resourceType).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRolesByResourceType 根据资源类型获取角色列表
func (r *RoleRepository) GetRolesByResourceType(ctx context.Context, resourceType string) ([]*model.Role, error) {
	var roles []*model.Role
	err := r.db.WithContext(ctx).Where("resource_type = ?", resourceType).Order("is_system DESC, created_at ASC").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetAllRoles 获取所有角色
func (r *RoleRepository) GetAllRoles(ctx context.Context) ([]*model.Role, error) {
	var roles []*model.Role
	err := r.db.WithContext(ctx).Order("is_system DESC, created_at ASC").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// UpdateRole 更新角色
func (r *RoleRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// DeleteRole 删除角色（系统角色不能删除）
func (r *RoleRepository) DeleteRole(ctx context.Context, id int64) error {
	// 检查是否为系统角色
	role, err := r.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return gorm.ErrRecordNotFound // 系统角色不能删除
	}
	return r.db.WithContext(ctx).Delete(&model.Role{}, id).Error
}

// CountRoles 统计角色数量
func (r *RoleRepository) CountRoles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Role{}).Count(&count).Error
	return count, err
}
