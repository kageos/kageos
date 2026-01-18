package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// RoleService 角色服务
type RoleService struct {
	roleRepo           *permissionrepo.RoleRepository
	rolePermissionRepo *permissionrepo.RolePermissionRepository
	roleAssignmentRepo *permissionrepo.RoleAssignmentRepository
	actionRepo         *permissionrepo.ActionRepository
	roleCache          *RoleCache
}

// NewRoleService 创建角色服务
func NewRoleService(
	roleRepo *permissionrepo.RoleRepository,
	rolePermissionRepo *permissionrepo.RolePermissionRepository,
	roleAssignmentRepo *permissionrepo.RoleAssignmentRepository,
	actionRepo *permissionrepo.ActionRepository,
	roleCache *RoleCache,
) *RoleService {
	return &RoleService{
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		roleAssignmentRepo: roleAssignmentRepo,
		actionRepo:         actionRepo,
		roleCache:          roleCache,
	}
}

// CreateRole 创建角色
// ⭐ 支持配置多个资源类型的权限点（如目录开发者可以配置目录权限 + 函数权限）
// 角色的 ResourceType 字段表示角色的主要资源类型（用于角色分组和查询）
func (s *RoleService) CreateRole(ctx context.Context, req *dto.CreateRoleReq) (*dto.CreateRoleResp, error) {
	// 1. ⭐ 从 Permissions 中推断主要 ResourceType（用于角色分组）
	// 如果配置了多个资源类型，使用第一个作为主要资源类型
	var primaryResourceType string
	for rt := range req.Permissions {
		if primaryResourceType == "" {
			primaryResourceType = rt
		}
		break // 只取第一个
	}
	if primaryResourceType == "" {
		return nil, fmt.Errorf("必须至少配置一个资源类型的权限")
	}

	// 2. 检查角色编码是否已存在（在同一资源类型中）
	existing, err := s.roleRepo.GetRoleByCode(ctx, req.Code, primaryResourceType)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("该资源类型中角色编码已存在: resourceType=%s, code=%s", primaryResourceType, req.Code)
	}

	// 3. 创建角色（主要资源类型用于角色分组和查询）
	role := &model.Role{
		Name:         req.Name,
		Code:         req.Code,
		ResourceType: primaryResourceType, // ⭐ 主要资源类型（用于角色分组）
		Description:  req.Description,
		IsSystem:     false, // 用户创建的角色不是系统角色
		CreatedBy:    contextx.GetRequestUser(ctx),
	}

	if err := s.roleRepo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	// 4. ⭐ 添加角色权限（支持配置多个资源类型的权限点）
	// 权限点格式：resource_type:action_type（如 directory:read, table:write）
	// 目录开发者可以配置多个资源类型的权限点（目录权限 + 函数权限）
	for permResourceType, actions := range req.Permissions {
		for _, actionCode := range actions {
			// 解析权限点编码，获取资源类型（用于索引）
			actionResourceType, _, ok := permissionpkg.ParseActionCode(actionCode)
			if !ok {
				// 如果不是新格式，尝试构建（向后兼容）
				actionCode = permissionpkg.BuildActionCode(permResourceType, actionCode)
				actionResourceType = permResourceType
			}

			// ⭐ 通过 ActionCode 查询 ActionID（建立外键关联）
			action, err := s.actionRepo.GetActionByCode(ctx, actionCode)
			if err != nil {
				return nil, fmt.Errorf("权限点不存在: code=%s, %w", actionCode, err)
			}

			rolePerm := &model.RolePermission{
				RoleID:   role.ID,
				ActionID: action.ID, // ⭐ 外键关联到 Action 表
			}
			if err := s.rolePermissionRepo.CreateRolePermission(ctx, rolePerm); err != nil {
				return nil, fmt.Errorf("添加角色权限失败: resourceType=%s, action=%s, %w", actionResourceType, actionCode, err)
			}
		}
	}

	// 4. ⭐ 刷新内存缓存
	if err := s.roleCache.Refresh(ctx); err != nil {
		logger.Warnf(ctx, "[RoleService] 刷新缓存失败: %v", err)
		// 不返回错误，因为角色已创建成功
	}

	logger.Infof(ctx, "[RoleService] 创建角色成功: code=%s, name=%s", role.Code, role.Name)

	return &dto.CreateRoleResp{
		Role: role,
	}, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(ctx context.Context, roleID int64, req *dto.UpdateRoleReq) (*dto.UpdateRoleResp, error) {
	// 1. 获取角色
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("角色不存在: %w", err)
	}

	// 2. 更新角色信息
	// ⭐ 系统预设角色支持修改所有字段（名称、描述、权限、默认值等）
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.IsDefault != nil {
		// ⭐ 如果设置为默认角色，需要先取消同资源类型的其他默认角色
		if *req.IsDefault {
			// 查询同资源类型的其他默认角色
			allRoles, err := s.roleRepo.GetRolesByResourceType(ctx, role.ResourceType)
			if err == nil {
				for _, r := range allRoles {
					if r.ID != roleID && r.IsDefault {
						r.IsDefault = false
						if err := s.roleRepo.UpdateRole(ctx, r); err != nil {
							logger.Warnf(ctx, "[RoleService] 取消其他默认角色失败: role_id=%d, %v", r.ID, err)
						}
					}
				}
			}
		}
		role.IsDefault = *req.IsDefault
	}

	if err := s.roleRepo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	// 4. 更新角色权限（如果提供了权限列表）
	if req.Permissions != nil {
		// 删除旧权限
		if err := s.rolePermissionRepo.DeleteByRoleID(ctx, roleID); err != nil {
			return nil, fmt.Errorf("删除旧权限失败: %w", err)
		}

		// ⭐ 添加新权限（支持配置多个资源类型的权限点）
		// 权限点格式：resource_type:action_type（如 directory:read, table:write）
		for permResourceType, actions := range *req.Permissions {
			for _, actionCode := range actions {
				// 解析权限点编码，获取资源类型（用于索引）
				actionResourceType, _, ok := permissionpkg.ParseActionCode(actionCode)
				if !ok {
					// 如果不是新格式，尝试构建（向后兼容）
					actionCode = permissionpkg.BuildActionCode(permResourceType, actionCode)
					actionResourceType = permResourceType
				}

				// ⭐ 通过 ActionCode 查询 ActionID（建立外键关联）
				action, err := s.actionRepo.GetActionByCode(ctx, actionCode)
				if err != nil {
					return nil, fmt.Errorf("权限点不存在: code=%s, %w", actionCode, err)
				}

				rolePerm := &model.RolePermission{
					RoleID:   roleID,
					ActionID: action.ID, // ⭐ 外键关联到 Action 表
				}
				if err := s.rolePermissionRepo.CreateRolePermission(ctx, rolePerm); err != nil {
					return nil, fmt.Errorf("添加角色权限失败: resourceType=%s, action=%s, %w", actionResourceType, actionCode, err)
				}
			}
		}
	}

	// 5. ⭐ 刷新内存缓存
	if err := s.roleCache.Refresh(ctx); err != nil {
		logger.Warnf(ctx, "[RoleService] 刷新缓存失败: %v", err)
	}

	logger.Infof(ctx, "[RoleService] 更新角色成功: role_id=%d", roleID)

	return &dto.UpdateRoleResp{
		Role: role,
	}, nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, roleID int64) error {
	// 1. 检查角色是否存在
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("角色不存在: %w", err)
	}

	// 2. 检查是否为系统角色（系统角色不能删除）
	if role.IsSystem {
		return fmt.Errorf("系统预设角色不能删除")
	}

	// 3. 删除角色权限
	if err := s.rolePermissionRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return fmt.Errorf("删除角色权限失败: %w", err)
	}

	// 4. 删除角色
	if err := s.roleRepo.DeleteRole(ctx, roleID); err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}

	// 5. ⭐ 刷新内存缓存
	if err := s.roleCache.Refresh(ctx); err != nil {
		logger.Warnf(ctx, "[RoleService] 刷新缓存失败: %v", err)
	}

	logger.Infof(ctx, "[RoleService] 删除角色成功: role_id=%d", roleID)

	return nil
}

// GetRole 获取角色
func (s *RoleService) GetRole(ctx context.Context, roleID int64) (*dto.GetRoleResp, error) {
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("角色不存在: %w", err)
	}

	// 加载权限列表
	perms, err := s.rolePermissionRepo.GetPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("获取角色权限失败: %w", err)
	}

	// 转换为模型格式（已经是 *RolePermission 类型，直接使用）
	role.Permissions = perms

	return &dto.GetRoleResp{
		Role: role,
	}, nil
}

// GetRoles 获取所有角色（从缓存读取）
func (s *RoleService) GetRoles(ctx context.Context, resourceType string) (*dto.GetRolesResp, error) {
	// ⭐ 从内存缓存读取所有角色
	cachedRoles := s.roleCache.GetAllRoles()

	// 转换为模型格式
	roles := make([]*model.Role, 0, len(cachedRoles))
	for _, cachedRole := range cachedRoles {
		// ⭐ 如果指定了资源类型，只返回该资源类型的角色
		if resourceType != "" && cachedRole.ResourceType != resourceType {
			continue
		}

		role := &model.Role{
			Name:         cachedRole.Name,
			Code:         cachedRole.Code,
			ResourceType: cachedRole.ResourceType, // ⭐ 保存资源类型
			Description:  cachedRole.Description,
			IsSystem:     cachedRole.IsSystem,
			IsDefault:    cachedRole.IsDefault, // ⭐ 默认角色标记
			CreatedBy:    cachedRole.CreatedBy, // ⭐ 创建者
		}
		role.ID = cachedRole.ID
		role.CreatedAt = cachedRole.CreatedAt // ⭐ 创建时间
		role.UpdatedAt = cachedRole.UpdatedAt // ⭐ 更新时间

		// ⭐ 从缓存中获取该角色的所有权限（包括所有资源类型的权限）
		// 例如：目录开发者角色可能包含 directory:* 和 table:*、form:*、chart:* 权限
		// ⭐ 从数据库查询完整的 RolePermission 记录（包含 ActionModel 关联）
		dbPerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(ctx, cachedRole.ID)
		if err == nil {
			role.Permissions = dbPerms
		} else {
			logger.Warnf(ctx, "[RoleService] 查询角色权限失败: role_id=%d, %v", cachedRole.ID, err)
			role.Permissions = []*model.RolePermission{}
		}

		roles = append(roles, role)
	}

	return &dto.GetRolesResp{
		Roles: roles,
	}, nil
}

// AssignRoleToUser 给用户分配角色
func (s *RoleService) AssignRoleToUser(ctx context.Context, req *dto.AssignRoleToUserReq) (*dto.AssignRoleToUserResp, error) {
	// 1. ⭐ 检查角色是否存在（需要同时匹配 code 和 resourceType）
	roleID, exists := s.roleCache.GetRoleIDByCode(req.RoleCode, req.ResourceType)
	if !exists {
		return nil, fmt.Errorf("角色不存在: resourceType=%s, code=%s", req.ResourceType, req.RoleCode)
	}

	// 2. 从 resource_path 解析 user 和 app（验证）
	_, user, app := permissionpkg.ParseFullCodePath(req.ResourcePath)
	if user == "" || app == "" {
		return nil, fmt.Errorf("无法从资源路径解析 user 和 app: %s", req.ResourcePath)
	}

	// 3. 验证 user 和 app 是否匹配
	if user != req.User || app != req.App {
		return nil, fmt.Errorf("资源路径中的 user 和 app 与请求参数不匹配")
	}

	// 4. 创建用户角色分配
	startTime := models.Time(time.Now())
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	assignment := &model.RoleAssignment{
		User:         req.User,
		App:          req.App,
		SubjectType:  "user",
		Subject:      req.Username,
		RoleID:       roleID,
		ResourcePath: req.ResourcePath,
		StartTime:    startTime,
		EndTime:      req.EndTime,
		CreatedBy:    contextx.GetRequestUser(ctx),
	}

	logger.Infof(ctx, "[RoleService] 创建用户角色分配: user=%s, app=%s, username=%s, role_code=%s, role_id=%d, resource_path=%s, start_time=%v, end_time=%v",
		req.User, req.App, req.Username, req.RoleCode, roleID, req.ResourcePath, startTime, req.EndTime)

	if err := s.roleAssignmentRepo.CreateRoleAssignment(ctx, assignment); err != nil {
		logger.Errorf(ctx, "[RoleService] 创建用户角色分配失败: user=%s, app=%s, username=%s, role_code=%s, error=%v",
			req.User, req.App, req.Username, req.RoleCode, err)
		return nil, fmt.Errorf("分配角色失败: %w", err)
	}

	logger.Infof(ctx, "[RoleService] 给用户分配角色成功: user=%s, app=%s, username=%s, role=%s, resource=%s, assignment_id=%d",
		req.User, req.App, req.Username, req.RoleCode, req.ResourcePath, assignment.ID)

	return &dto.AssignRoleToUserResp{
		Assignment: assignment,
	}, nil
}

// AssignRoleToDepartment 给组织架构分配角色
func (s *RoleService) AssignRoleToDepartment(ctx context.Context, req *dto.AssignRoleToDepartmentReq) (*dto.AssignRoleToDepartmentResp, error) {
	// 1. ⭐ 检查角色是否存在（需要同时匹配 code 和 resourceType）
	roleID, exists := s.roleCache.GetRoleIDByCode(req.RoleCode, req.ResourceType)
	if !exists {
		return nil, fmt.Errorf("角色不存在: resourceType=%s, code=%s", req.ResourceType, req.RoleCode)
	}

	// 2. 从 resource_path 解析 user 和 app（验证）
	_, user, app := permissionpkg.ParseFullCodePath(req.ResourcePath)
	if user == "" || app == "" {
		return nil, fmt.Errorf("无法从资源路径解析 user 和 app: %s", req.ResourcePath)
	}

	// 3. 验证 user 和 app 是否匹配
	if user != req.User || app != req.App {
		return nil, fmt.Errorf("资源路径中的 user 和 app 与请求参数不匹配")
	}

	// 4. 创建组织架构角色分配
	startTime := models.Time(time.Now())
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	assignment := &model.RoleAssignment{
		User:         req.User,
		App:          req.App,
		SubjectType:  "department",
		Subject:      req.DepartmentPath,
		RoleID:       roleID,
		ResourcePath: req.ResourcePath,
		StartTime:    startTime,
		EndTime:      req.EndTime,
		CreatedBy:    contextx.GetRequestUser(ctx),
	}

	logger.Infof(ctx, "[RoleService] 创建组织架构角色分配: user=%s, app=%s, dept=%s, role_code=%s, role_id=%d, resource_path=%s, start_time=%v, end_time=%v",
		req.User, req.App, req.DepartmentPath, req.RoleCode, roleID, req.ResourcePath, startTime, req.EndTime)

	if err := s.roleAssignmentRepo.CreateRoleAssignment(ctx, assignment); err != nil {
		logger.Errorf(ctx, "[RoleService] 创建组织架构角色分配失败: user=%s, app=%s, dept=%s, role_code=%s, error=%v",
			req.User, req.App, req.DepartmentPath, req.RoleCode, err)
		return nil, fmt.Errorf("分配角色失败: %w", err)
	}

	logger.Infof(ctx, "[RoleService] 给组织架构分配角色成功: user=%s, app=%s, dept=%s, role=%s, resource=%s, assignment_id=%d",
		req.User, req.App, req.DepartmentPath, req.RoleCode, req.ResourcePath, assignment.ID)

	return &dto.AssignRoleToDepartmentResp{
		Assignment: assignment,
	}, nil
}

// RemoveRoleFromUser 移除用户角色
func (s *RoleService) RemoveRoleFromUser(ctx context.Context, req *dto.RemoveRoleFromUserReq) error {
	// 1. ⭐ 检查角色是否存在（需要同时匹配 code 和 resourceType）
	roleID, exists := s.roleCache.GetRoleIDByCode(req.RoleCode, req.ResourceType)
	if !exists {
		return fmt.Errorf("角色不存在: resourceType=%s, code=%s", req.ResourceType, req.RoleCode)
	}

	// 2. 删除角色分配
	if err := s.roleAssignmentRepo.DeleteRoleAssignmentByUser(ctx, req.User, req.App, req.Username, roleID, req.ResourcePath); err != nil {
		return fmt.Errorf("移除角色失败: %w", err)
	}

	logger.Infof(ctx, "[RoleService] 移除用户角色成功: user=%s, app=%s, username=%s, role=%s, resource=%s",
		req.User, req.App, req.Username, req.RoleCode, req.ResourcePath)

	return nil
}

// RemoveRoleFromDepartment 移除组织架构角色
func (s *RoleService) RemoveRoleFromDepartment(ctx context.Context, req *dto.RemoveRoleFromDepartmentReq) error {
	// 1. ⭐ 检查角色是否存在（需要同时匹配 code 和 resourceType）
	roleID, exists := s.roleCache.GetRoleIDByCode(req.RoleCode, req.ResourceType)
	if !exists {
		return fmt.Errorf("角色不存在: resourceType=%s, code=%s", req.ResourceType, req.RoleCode)
	}

	// 2. 删除角色分配
	if err := s.roleAssignmentRepo.DeleteRoleAssignmentByDepartment(ctx, req.User, req.App, req.DepartmentPath, roleID, req.ResourcePath); err != nil {
		return fmt.Errorf("移除角色失败: %w", err)
	}

	logger.Infof(ctx, "[RoleService] 移除组织架构角色成功: user=%s, app=%s, dept=%s, role=%s, resource=%s",
		req.User, req.App, req.DepartmentPath, req.RoleCode, req.ResourcePath)

	return nil
}

// GetUserRoles 获取用户角色
func (s *RoleService) GetUserRoles(ctx context.Context, req *dto.GetUserRolesReq) (*dto.GetUserRolesResp, error) {
	assignments, err := s.roleAssignmentRepo.GetRoleAssignmentsByUser(ctx, req.User, req.App, req.Username)
	if err != nil {
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	return &dto.GetUserRolesResp{
		Assignments: assignments,
	}, nil
}

// GetDepartmentRoles 获取组织架构角色
func (s *RoleService) GetDepartmentRoles(ctx context.Context, req *dto.GetDepartmentRolesReq) (*dto.GetDepartmentRolesResp, error) {
	assignments, err := s.roleAssignmentRepo.GetRoleAssignmentsByDepartment(ctx, req.User, req.App, req.DepartmentPath)
	if err != nil {
		return nil, fmt.Errorf("获取组织架构角色失败: %w", err)
	}

	return &dto.GetDepartmentRolesResp{
		Assignments: assignments,
	}, nil
}

// GetResourcePermissions 查询资源的所有权限分配
func (s *RoleService) GetResourcePermissions(ctx context.Context, req *dto.GetResourcePermissionsReq) (*dto.GetResourcePermissionsResp, error) {
	// 1. 查询角色分配
	assignments, err := s.roleAssignmentRepo.GetRoleAssignmentsByResourcePath(ctx, req.User, req.App, req.ResourcePath)
	if err != nil {
		return nil, fmt.Errorf("查询角色分配失败: %w", err)
	}

	// 2. 构建响应数据
	result := make([]*dto.ResourcePermissionAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		// 获取角色信息
		role, exists := s.roleCache.GetRole(assignment.RoleID)
		if !exists || role == nil {
			// 角色不存在，跳过
			logger.Warnf(ctx, "[RoleService] 角色不存在: role_id=%d", assignment.RoleID)
			continue
		}

		// 权限主体名称（暂时使用 subject 值，后续可以优化为查询用户/部门信息）
		subjectName := assignment.Subject
		// TODO: 可以调用 hr-server 接口批量查询用户/部门信息，获取显示名称

		// 使用 models.Time 类型（不需要格式化）
		var endTime *models.Time
		if assignment.EndTime != nil {
			endTime = assignment.EndTime
		}

		result = append(result, &dto.ResourcePermissionAssignment{
			ID:           assignment.ID,
			SubjectType:  assignment.SubjectType,
			Subject:      assignment.Subject,
			SubjectName:  subjectName,
			RoleID:       assignment.RoleID,
			RoleCode:     role.Code,
			RoleName:     role.Name,
			ResourcePath: assignment.ResourcePath,
			StartTime:    assignment.StartTime,
			EndTime:      endTime,
			CreatedBy:    assignment.CreatedBy,
			CreatedAt:    assignment.CreatedAt,
		})
	}

	return &dto.GetResourcePermissionsResp{
		Assignments: result,
		Total:       len(result),
	}, nil
}

// InitDefaultRoles 初始化预设角色
//
// ⭐ 角色和权限点的关系：
//   - 角色（Role）：按资源类型分组（directory、table、form、chart、app）
//   - 权限点（Action）：格式为 resource_type:action_type（如 table:read、form:write）
//   - 角色权限（RolePermission）：角色和权限点的关联关系（通过 ActionID 外键关联到 action 表）
//   - 一个角色可以配置多个权限点（支持跨资源类型，如目录开发者可以配置目录权限 + 函数权限）
//
// ⭐ 预设角色配置说明：
//   - 目录角色（directory）：可以配置目录权限 + 函数权限（table、form、chart）
//   - 目录查看者：配置了 directory:read + table:read + form:read + chart:read
//   - 目录开发者：配置了 directory:read/write/update + table:read/write/update/delete + form:read/write + chart:read
//   - 目录管理员：配置了 directory:admin + table:admin + form:admin + chart:admin
//   - 函数角色（table、form、chart）：只配置对应资源类型的权限点
//   - 表格查看者：只配置 table:read
//   - 表格开发者：配置 table:read/write/update/delete
//   - 表格管理员：配置 table:admin
//
// ⭐ 权限继承说明：
//   - 目录角色配置的函数权限（如 table:write）会被子函数继承（方式2：直接函数权限继承）
//   - 例如：父目录配置了目录开发者角色（包含 table:write），子函数自动继承 table:write 权限
//   - 不需要转换，直接匹配相同的权限点
//
// ⭐ 如果角色已存在，会更新系统角色的权限以匹配最新配置
func (s *RoleService) InitDefaultRoles(ctx context.Context) error {
	// ⭐ 预设角色配置（使用权限点表管理，格式：resource_type:action_type）
	// 目录开发者可以配置多个资源类型的权限点（目录权限 + 函数权限）
	defaultRoles := []struct {
		resourceType string
		name         string
		code         string
		description  string
		isDefault    bool     // 是否默认角色（用于权限申请时的默认推荐）
		actions      []string // 权限点编码（如 directory:read, table:write）
	}{
		// Directory 资源类型的角色
		{
			resourceType: permissionpkg.ResourceTypeDirectory,
			name:         "查看者",
			code:         "viewer",
			description:  "目录查看者，拥有查看目录和目录下函数的权限",
			isDefault:    true, // ⭐ 默认角色
			actions: []string{
				// 目录权限
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeDirectory, "read"),
				// 函数权限（目录查看者可以查看所有类型的函数）
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeChart, "read"),
			},
		},
		{
			resourceType: permissionpkg.ResourceTypeDirectory,
			name:         "开发者",
			code:         "developer",
			description:  "目录开发者，拥有查看、创建和修改目录、函数的权限",
			isDefault:    false,
			// ⭐ 目录开发者可以配置多个资源类型的权限点
			actions: []string{
				// 目录权限
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeDirectory, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeDirectory, "write"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeDirectory, "update"),
				// 函数权限（目录开发者可以操作函数）
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "write"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "update"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "delete"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "write"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeChart, "read"),
			},
		},
		{
			resourceType: permissionpkg.ResourceTypeDirectory,
			name:         "管理员",
			code:         "admin",
			description:  "目录管理员，拥有完整的管理权限（包括目录和所有函数类型）",
			isDefault:    false,
			actions: []string{
				// 目录权限
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeDirectory, "admin"),
				// 函数权限（目录管理员拥有所有函数类型的所有权限）
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "admin"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "admin"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeChart, "admin"),
			},
		},
		// Table 资源类型的角色
		{
			resourceType: permissionpkg.ResourceTypeTable,
			name:         "查看者",
			code:         "viewer",
			description:  "表格查看者，拥有查看表格的权限",
			isDefault:    true, // ⭐ 默认角色
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "read")},
		},
		{
			resourceType: permissionpkg.ResourceTypeTable,
			name:         "开发者",
			code:         "developer",
			description:  "表格开发者，拥有查看、创建、修改、删除表格的权限",
			isDefault:    false,
			actions: []string{
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "write"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "update"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "delete"),
			},
		},
		{
			resourceType: permissionpkg.ResourceTypeTable,
			name:         "管理员",
			code:         "admin",
			description:  "表格管理员，拥有完整的管理权限",
			isDefault:    false,
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeTable, "admin")},
		},
		// Form 资源类型的角色
		{
			resourceType: permissionpkg.ResourceTypeForm,
			name:         "查看者",
			code:         "viewer",
			description:  "表单查看者，拥有查看表单的权限",
			isDefault:    true, // ⭐ 默认角色
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "read")},
		},
		{
			resourceType: permissionpkg.ResourceTypeForm,
			name:         "开发者",
			code:         "developer",
			description:  "表单开发者，拥有查看和提交表单的权限",
			isDefault:    false,
			actions: []string{
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "read"),
				permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "write"),
			},
		},
		{
			resourceType: permissionpkg.ResourceTypeForm,
			name:         "管理员",
			code:         "admin",
			description:  "表单管理员，拥有完整的管理权限",
			isDefault:    false,
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeForm, "admin")},
		},
		// Chart 资源类型的角色
		{
			resourceType: permissionpkg.ResourceTypeChart,
			name:         "查看者",
			code:         "viewer",
			description:  "图表查看者，拥有查看图表的权限",
			isDefault:    true, // ⭐ 默认角色
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeChart, "read")},
		},
		{
			resourceType: permissionpkg.ResourceTypeChart,
			name:         "管理员",
			code:         "admin",
			description:  "图表管理员，拥有完整的管理权限",
			isDefault:    false,
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeChart, "admin")},
		},
		// App 资源类型的角色
		{
			resourceType: permissionpkg.ResourceTypeApp,
			name:         "管理员",
			code:         "admin",
			description:  "工作空间管理员，拥有完整的管理权限",
			isDefault:    false,
			actions:      []string{permissionpkg.BuildActionCode(permissionpkg.ResourceTypeApp, "admin")},
		},
	}

	// 创建或更新预设角色（每个资源类型独立处理）
	for _, roleConfig := range defaultRoles {
		// ⭐ 检查角色是否已存在（根据 resourceType 和 code 查找）
		roleID, exists := s.roleCache.GetRoleIDByCode(roleConfig.code, roleConfig.resourceType)
		var role *model.Role
		var err error

		if exists {
			// 角色已存在，获取并更新
			role, err = s.roleRepo.GetRoleByID(ctx, roleID)
			if err != nil {
				return fmt.Errorf("获取预设角色失败: resourceType=%s, code=%s, %w", roleConfig.resourceType, roleConfig.code, err)
			}

			// 更新角色信息（仅更新描述和 isDefault，名称和 code 不变）
			role.Description = roleConfig.description
			role.IsDefault = roleConfig.isDefault
			if err := s.roleRepo.UpdateRole(ctx, role); err != nil {
				return fmt.Errorf("更新预设角色失败: resourceType=%s, code=%s, %w", roleConfig.resourceType, roleConfig.code, err)
			}

			// ⭐ 删除旧权限，重新添加（确保权限与最新配置一致）
			if err := s.rolePermissionRepo.DeleteByRoleID(ctx, role.ID); err != nil {
				return fmt.Errorf("删除预设角色旧权限失败: resourceType=%s, code=%s, %w", roleConfig.resourceType, roleConfig.code, err)
			}

			logger.Infof(ctx, "[RoleService] 更新预设角色: resourceType=%s, code=%s, name=%s", roleConfig.resourceType, roleConfig.code, role.Name)
		} else {
			// 角色不存在，创建新角色
			now := models.Time(time.Now())
			role = &model.Role{
				Base: models.Base{
					CreatedAt: now, // ⭐ 显式设置创建时间
					UpdatedAt: now, // ⭐ 显式设置更新时间
					CreatedBy: "system",
				},
				Name:         roleConfig.name,
				Code:         roleConfig.code,
				ResourceType: roleConfig.resourceType, // ⭐ 角色绑定到特定资源类型
				Description:  roleConfig.description,
				IsSystem:     true,                 // 系统预设角色
				IsDefault:    roleConfig.isDefault, // ⭐ 默认角色标记
			}

			if err := s.roleRepo.CreateRole(ctx, role); err != nil {
				return fmt.Errorf("创建预设角色失败: resourceType=%s, code=%s, %w", roleConfig.resourceType, roleConfig.code, err)
			}

			logger.Infof(ctx, "[RoleService] 创建预设角色成功: resourceType=%s, code=%s, name=%s", roleConfig.resourceType, roleConfig.code, role.Name)
		}

		// 添加角色权限（支持多个资源类型的权限点）
		// ⭐ 从 Action 字段解析资源类型（格式：resource_type:action_type）
		for _, action := range roleConfig.actions {
			// 解析权限点编码，获取资源类型
			actionResourceType, _, ok := permissionpkg.ParseActionCode(action)
			if !ok {
				// 如果解析失败，使用角色配置的资源类型（向后兼容）
				actionResourceType = roleConfig.resourceType
			}

			// ⭐ 通过 ActionCode 查询 ActionID（建立外键关联）
			actionModel, err := s.actionRepo.GetActionByCode(ctx, action)
			if err != nil {
				return fmt.Errorf("权限点不存在: code=%s, %w", action, err)
			}

			rolePerm := &model.RolePermission{
				RoleID:   role.ID,
				ActionID: actionModel.ID, // ⭐ 外键关联到 Action 表
			}
			if err := s.rolePermissionRepo.CreateRolePermission(ctx, rolePerm); err != nil {
				return fmt.Errorf("创建预设角色权限失败: resourceType=%s, code=%s, action=%s, %w", actionResourceType, roleConfig.code, action, err)
			}
		}
	}

	// ⭐ 刷新内存缓存
	if err := s.roleCache.Refresh(ctx); err != nil {
		logger.Warnf(ctx, "[RoleService] 刷新缓存失败: %v", err)
	}

	logger.Infof(ctx, "[RoleService] 预设角色初始化完成")
	return nil
}

// GetRolesForPermissionRequest 获取可用于权限申请的角色列表（根据节点类型过滤）
// ⭐ 支持返回该资源类型的角色，以及目录开发者角色（如果目录开发者配置了该资源类型的权限）
func (s *RoleService) GetRolesForPermissionRequest(ctx context.Context, req *dto.GetRolesForPermissionRequestReq) (*dto.GetRolesForPermissionRequestResp, error) {
	// 1. 根据节点类型和模板类型获取资源类型
	resourceType := permissionpkg.GetResourceType(req.NodeType, req.TemplateType)
	if resourceType == "" {
		return nil, fmt.Errorf("无效的节点类型或模板类型: nodeType=%s, templateType=%s", req.NodeType, req.TemplateType)
	}

	// 2. ⭐ 只查询该资源类型的角色（函数节点只显示对应资源类型的角色）
	roles, err := s.roleRepo.GetRolesByResourceType(ctx, resourceType)
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	// 3. 从缓存获取角色权限信息，构建完整的角色模型
	filteredRoles := make([]*model.Role, 0)
	for _, role := range roles {
		// 从缓存获取该角色的权限（只包含该资源类型的权限）
		perms := s.roleCache.GetRolePermissionsByResourceType(role.ID, resourceType)
		if len(perms) == 0 {
			// 该角色没有该资源类型的权限，跳过
			continue
		}

		// ⭐ 从数据库查询该资源类型的权限（包含 ActionModel 关联）
		rolePerms, err := s.rolePermissionRepo.GetPermissionsByRoleIDAndResourceType(ctx, role.ID, resourceType)
		if err == nil {
			role.Permissions = rolePerms
		} else {
			logger.Warnf(ctx, "[RoleService] 查询角色权限失败: role_id=%d, resource_type=%s, %v", role.ID, resourceType, err)
			role.Permissions = []*model.RolePermission{}
		}

		filteredRoles = append(filteredRoles, role)
	}

	logger.Debugf(ctx, "[RoleService] 获取可用于权限申请的角色列表: resourceType=%s, 角色数=%d",
		resourceType, len(filteredRoles))

	return &dto.GetRolesForPermissionRequestResp{
		Roles: filteredRoles,
	}, nil
}
