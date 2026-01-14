package repository

import (
	"context"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// RoleAssignmentRepository 角色分配仓储
type RoleAssignmentRepository struct {
	db *gorm.DB
}

// NewRoleAssignmentRepository 创建角色分配仓储
func NewRoleAssignmentRepository(db *gorm.DB) *RoleAssignmentRepository {
	return &RoleAssignmentRepository{db: db}
}

// SubjectInfo 权限主体信息
type SubjectInfo struct {
	Type  string
	Value string
}

// CreateRoleAssignment 创建角色分配
func (r *RoleAssignmentRepository) CreateRoleAssignment(ctx context.Context, assignment *model.RoleAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

// GetRoleAssignmentsBySubjects 根据权限主体列表查询角色分配
// ⭐ 一次查询获取所有角色分配（用户 + 组织架构）
func (r *RoleAssignmentRepository) GetRoleAssignmentsBySubjects(
	ctx context.Context,
	user string,
	app string,
	subjects []SubjectInfo,
) ([]*model.RoleAssignment, error) {
	if len(subjects) == 0 {
		return []*model.RoleAssignment{}, nil
	}
	
	if user == "" || app == "" {
		return []*model.RoleAssignment{}, nil
	}
	
	var assignments []*model.RoleAssignment
	query := r.db.WithContext(ctx).Where("user = ? AND app = ?", user, app)
	
	// 构建 (subject_type, subject) 的查询条件
	var conditions []string
	var args []interface{}
	
	for _, subject := range subjects {
		conditions = append(conditions, "(subject_type = ? AND subject = ?)")
		args = append(args, subject.Type, subject.Value)
	}
	
	if len(conditions) > 0 {
		query = query.Where("("+joinRoleAssignmentStrings(conditions, " OR ")+")", args...)
	}
	
	// 只查询未过期的角色分配
	now := time.Now()
	query = query.Where("start_time <= ?", now).
		Where("(end_time IS NULL OR end_time > ?)", now)
	
	err := query.Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	
	return assignments, nil
}

// GetRoleAssignmentsByUser 根据用户查询角色分配
func (r *RoleAssignmentRepository) GetRoleAssignmentsByUser(ctx context.Context, user, app, username string) ([]*model.RoleAssignment, error) {
	var assignments []*model.RoleAssignment
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("user = ? AND app = ? AND subject_type = ? AND subject = ?", user, app, "user", username).
		Where("start_time <= ?", now).
		Where("(end_time IS NULL OR end_time > ?)", now).
		Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetRoleAssignmentsByDepartment 根据组织架构查询角色分配
func (r *RoleAssignmentRepository) GetRoleAssignmentsByDepartment(ctx context.Context, user, app, departmentPath string) ([]*model.RoleAssignment, error) {
	var assignments []*model.RoleAssignment
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("user = ? AND app = ? AND subject_type = ? AND subject = ?", user, app, "department", departmentPath).
		Where("start_time <= ?", now).
		Where("(end_time IS NULL OR end_time > ?)", now).
		Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// DeleteRoleAssignment 删除角色分配
func (r *RoleAssignmentRepository) DeleteRoleAssignment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.RoleAssignment{}, id).Error
}

// DeleteRoleAssignmentByUser 删除用户的角色分配
func (r *RoleAssignmentRepository) DeleteRoleAssignmentByUser(ctx context.Context, user, app, username string, roleID int64, resourcePath string) error {
	return r.db.WithContext(ctx).
		Where("user = ? AND app = ? AND subject_type = ? AND subject = ? AND role_id = ? AND resource_path = ?",
			user, app, "user", username, roleID, resourcePath).
		Delete(&model.RoleAssignment{}).Error
}

// DeleteRoleAssignmentByDepartment 删除组织架构的角色分配
func (r *RoleAssignmentRepository) DeleteRoleAssignmentByDepartment(ctx context.Context, user, app, departmentPath string, roleID int64, resourcePath string) error {
	return r.db.WithContext(ctx).
		Where("user = ? AND app = ? AND subject_type = ? AND subject = ? AND role_id = ? AND resource_path = ?",
			user, app, "department", departmentPath, roleID, resourcePath).
		Delete(&model.RoleAssignment{}).Error
}

// joinRoleAssignmentStrings 辅助函数：用分隔符连接字符串（避免与 permission_repository 中的 joinStrings 冲突）
func joinRoleAssignmentStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
