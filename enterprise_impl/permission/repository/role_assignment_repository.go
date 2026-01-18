package repository

import (
	"context"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
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

// GetRoleAssignmentsByResourcePath 根据资源路径查询角色分配
// ⭐ 只返回当前节点和父节点的权限，不返回子节点的权限
//   1. 精确匹配：resource_path = /a/b/c （当前节点）
//   2. 向上继承：resource_path = /a/b 或 /a （父目录权限，/a/b/c 继承 /a/b 和 /a 的权限）
func (r *RoleAssignmentRepository) GetRoleAssignmentsByResourcePath(ctx context.Context, user string, app string, resourcePath string) ([]*model.RoleAssignment, error) {
	if user == "" || app == "" || resourcePath == "" {
		return []*model.RoleAssignment{}, nil
	}

	var assignments []*model.RoleAssignment

	// 构建查询条件
	// 1. 精确匹配：resource_path = /a/b/c （当前节点）
	exactMatch := resourcePath

	// 2. 向上继承：获取所有父路径
	// 例如：/a/b/c -> [/a/b, /a]
	parentPaths := getAllParentPaths(resourcePath)

	// 构建查询条件
	var conditions []string
	var args []interface{}

	// 精确匹配（当前节点）
	conditions = append(conditions, "resource_path = ?")
	args = append(args, exactMatch)

	// 向上继承（父路径精确匹配）
	for _, parentPath := range parentPaths {
		conditions = append(conditions, "resource_path = ?")
		args = append(args, parentPath)
	}

	// 执行查询
	// ⭐ 权限管理需要显示所有权限分配（包括未生效和已过期的），不进行时间过滤
	// 权限计算（用于判断用户是否有权限）才需要只查询生效的权限
	query := r.db.WithContext(ctx).
		Where("user = ? AND app = ?", user, app).
		Where("("+joinRoleAssignmentStrings(conditions, " OR ")+")", args...).
		Preload("Role").
		Order("created_at DESC")
	
	logger.Infof(ctx, "[RoleAssignmentRepository] 查询资源权限: user=%s, app=%s, resourcePath=%s, conditions=%v, args=%v",
		user, app, resourcePath, conditions, args)
	
	err := query.Find(&assignments).Error

	if err != nil {
		logger.Errorf(ctx, "[RoleAssignmentRepository] 查询资源权限失败: user=%s, app=%s, resourcePath=%s, error=%v",
			user, app, resourcePath, err)
		return nil, err
	}

	logger.Infof(ctx, "[RoleAssignmentRepository] 查询资源权限结果: user=%s, app=%s, resourcePath=%s, count=%d",
		user, app, resourcePath, len(assignments))

	return assignments, nil
}

// getAllParentPaths 获取资源路径的所有父路径
// 例如：/a/b/c -> [/a/b, /a]
//      /user/app/dir/function -> [/user/app/dir, /user/app, /user]
func getAllParentPaths(resourcePath string) []string {
	if resourcePath == "" || resourcePath == "/" {
		return []string{}
	}

	// 移除开头的 /
	path := strings.TrimPrefix(resourcePath, "/")
	parts := strings.Split(path, "/")

	if len(parts) <= 1 {
		return []string{}
	}

	// 构建父路径列表
	parentPaths := make([]string, 0, len(parts)-1)
	for i := len(parts) - 1; i > 0; i-- {
		parentPath := "/" + strings.Join(parts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}

	return parentPaths
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
