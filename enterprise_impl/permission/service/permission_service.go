package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
	"gorm.io/gorm"
)

// PermissionServiceImpl 权限服务实现（企业版）
// ⭐ 完全移除 Casbin 和 workspace_permission 表，仅使用角色系统
type PermissionServiceImpl struct {
	db                    *gorm.DB
	permissionRequestRepo *permissionrepo.PermissionRequestRepository
	serviceTreeRepo       *repository.ServiceTreeRepository // ⭐ 使用社区版的 repository，不重复造轮子
	appRepo               *repository.AppRepository         // ⭐ 用于检查 app.Admins 字段
	approvalService       *ApprovalService                   // ⭐ 内部实现，不对外暴露
	permissionCalculator  PermissionCalculatorInterface      // ⭐ 权限计算器接口（支持新旧版本）
	// ⭐ 角色系统相关
	roleRepo              *permissionrepo.RoleRepository
	rolePermissionRepo    *permissionrepo.RolePermissionRepository
	roleAssignmentRepo    *permissionrepo.RoleAssignmentRepository
	roleCache             *RoleCache
	roleService           *RoleService
}

// NewPermissionService 创建权限服务（企业版）
func NewPermissionService(db *gorm.DB) (*PermissionServiceImpl, error) {
	return &PermissionServiceImpl{
		db: db,
	}, nil
}

// Init 初始化权限服务
func (s *PermissionServiceImpl) Init(opt *enterprise.InitOptions) error {
	s.db = opt.DB

	// 初始化仓储层（不再使用 permissionRepo，仅使用角色系统）
	s.permissionRequestRepo = permissionrepo.NewPermissionRequestRepository(opt.DB)
	s.serviceTreeRepo = repository.NewServiceTreeRepository(opt.DB) // ⭐ 使用社区版的 repository
	s.appRepo = repository.NewAppRepository(opt.DB)                 // ⭐ 用于检查 app.Admins 字段

	// ⭐ 初始化角色系统仓储
	s.roleRepo = permissionrepo.NewRoleRepository(opt.DB)
	s.rolePermissionRepo = permissionrepo.NewRolePermissionRepository(opt.DB)
	s.roleAssignmentRepo = permissionrepo.NewRoleAssignmentRepository(opt.DB)

	// ⭐ 初始化角色缓存
	ctx := context.Background()
	s.roleCache = NewRoleCache(s.roleRepo, s.rolePermissionRepo)
	if err := s.roleCache.LoadAllRoles(ctx); err != nil {
		logger.Warnf(ctx, "[PermissionService] 加载角色缓存失败: %v", err)
	}

	// ⭐ 初始化权限点服务
	actionRepo := permissionrepo.NewActionRepository(opt.DB)
	actionService := NewActionService(actionRepo)
	// ⭐ 初始化预设权限点（必须在初始化角色之前）
	if err := actionService.InitDefaultActions(ctx); err != nil {
		logger.Warnf(ctx, "[PermissionService] 初始化预设权限点失败: %v", err)
	}

	// ⭐ 初始化角色服务
	s.roleService = NewRoleService(
		s.roleRepo,
		s.rolePermissionRepo,
		s.roleAssignmentRepo,
		s.roleCache,
	)

	// ⭐ 初始化预设角色（必须在初始化权限点之后）
	if err := s.roleService.InitDefaultRoles(ctx); err != nil {
		logger.Warnf(ctx, "[PermissionService] 初始化预设角色失败: %v", err)
	}

	// 初始化审批服务（仅使用角色系统）
	s.approvalService = NewApprovalService(
		s.permissionRequestRepo,
		s.serviceTreeRepo,
		s.roleService, // ⭐ 传入角色服务，用于审批通过后分配角色
		s.roleCache,   // ⭐ 传入角色缓存，用于从 RoleID 获取 RoleCode
	)

	// ⭐ 初始化权限计算器 V2（结构化版本，仅使用角色系统）
	s.permissionCalculator = NewPermissionCalculatorV2(
		s.roleAssignmentRepo,
		s.roleCache,
	)

	logger.Infof(ctx, "[PermissionService] 新权限系统初始化完成（已移除 Casbin，已集成角色系统，角色和权限点已缓存）")

	return nil
}

// ============================================
// 角色管理方法实现
// ============================================

// GetRoles 获取所有角色
func (s *PermissionServiceImpl) GetRoles(ctx context.Context, resourceType string) (*dto.GetRolesResp, error) {
	return s.roleService.GetRoles(ctx, resourceType)
}

// GetRole 获取角色详情
func (s *PermissionServiceImpl) GetRole(ctx context.Context, roleID int64) (*dto.GetRoleResp, error) {
	return s.roleService.GetRole(ctx, roleID)
}

// CreateRole 创建角色
func (s *PermissionServiceImpl) CreateRole(ctx context.Context, req *dto.CreateRoleReq) (*dto.CreateRoleResp, error) {
	return s.roleService.CreateRole(ctx, req)
}

// UpdateRole 更新角色
func (s *PermissionServiceImpl) UpdateRole(ctx context.Context, roleID int64, req *dto.UpdateRoleReq) (*dto.UpdateRoleResp, error) {
	return s.roleService.UpdateRole(ctx, roleID, req)
}

// DeleteRole 删除角色
func (s *PermissionServiceImpl) DeleteRole(ctx context.Context, roleID int64) error {
	return s.roleService.DeleteRole(ctx, roleID)
}

// AssignRoleToUser 给用户分配角色
func (s *PermissionServiceImpl) AssignRoleToUser(ctx context.Context, req *dto.AssignRoleToUserReq) (*dto.AssignRoleToUserResp, error) {
	return s.roleService.AssignRoleToUser(ctx, req)
}

// AssignRoleToDepartment 给组织架构分配角色
func (s *PermissionServiceImpl) AssignRoleToDepartment(ctx context.Context, req *dto.AssignRoleToDepartmentReq) (*dto.AssignRoleToDepartmentResp, error) {
	return s.roleService.AssignRoleToDepartment(ctx, req)
}

// RemoveRoleFromUser 移除用户角色
func (s *PermissionServiceImpl) RemoveRoleFromUser(ctx context.Context, req *dto.RemoveRoleFromUserReq) error {
	return s.roleService.RemoveRoleFromUser(ctx, req)
}

// RemoveRoleFromDepartment 移除组织架构角色
func (s *PermissionServiceImpl) RemoveRoleFromDepartment(ctx context.Context, req *dto.RemoveRoleFromDepartmentReq) error {
	return s.roleService.RemoveRoleFromDepartment(ctx, req)
}

// GetUserRoles 获取用户角色
func (s *PermissionServiceImpl) GetUserRoles(ctx context.Context, req *dto.GetUserRolesReq) (*dto.GetUserRolesResp, error) {
	return s.roleService.GetUserRoles(ctx, req)
}

// GetDepartmentRoles 获取组织架构角色
func (s *PermissionServiceImpl) GetDepartmentRoles(ctx context.Context, req *dto.GetDepartmentRolesReq) (*dto.GetDepartmentRolesResp, error) {
	return s.roleService.GetDepartmentRoles(ctx, req)
}

// GetRolesForPermissionRequest 获取可用于权限申请的角色列表
func (s *PermissionServiceImpl) GetRolesForPermissionRequest(ctx context.Context, req *dto.GetRolesForPermissionRequestReq) (*dto.GetRolesForPermissionRequestResp, error) {
	return s.roleService.GetRolesForPermissionRequest(ctx, req)
}

// PermissionCheck 权限检查项
type PermissionCheck struct {
	ResourcePath string // 资源路径
	Action       string // 操作
	Priority     int    // 优先级（数字越小，优先级越高）
}

// CheckPermission 检查用户权限（企业版实现）
// ⭐ 使用 GetUserWorkspacePermissions 获取所有权限，然后在应用层校验
// 支持层级权限继承：自动向上查找父节点的权限
func (s *PermissionServiceImpl) CheckPermission(ctx context.Context, username string, resourcePath string, action string) (bool, error) {
	// 从 context 获取组织架构路径
	departmentPath := contextx.GetRequestDepartmentFullPath(ctx)

	// ⭐ 从 resourcePath 解析 user 和 app
	_, user, app := permissionpkg.ParseFullCodePath(resourcePath)
	if user == "" || app == "" {
		logger.Warnf(ctx, "[PermissionService] 无法从资源路径解析 user 和 app: resource=%s，返回 false", resourcePath)
		return false, nil
	}

	// ⭐ 优先检查：如果当前用户是工作空间管理员，直接返回 true
	if s.appRepo != nil {
		appModel, err := s.appRepo.GetAppByUserName(user, app)
		if err == nil && appModel != nil && appModel.Admins != "" {
			adminList := strings.Split(appModel.Admins, ",")
			for _, admin := range adminList {
				admin = strings.TrimSpace(admin)
				if admin == username {
					// 当前用户是管理员，直接返回 true（拥有所有权限）
					logger.Debugf(ctx, "[PermissionService] 用户 %s 是工作空间管理员，直接返回 true", username)
					return true, nil
				}
			}
		}
	}

	// ⭐ 如果不是管理员，使用原有的权限查询逻辑
	// ⭐ 使用 GetUserWorkspacePermissions 获取所有权限记录
	req := &enterprise.GetUserWorkspacePermissionsReq{
		User:           user,
		App:            app,
		Username:       username,
		DepartmentPath: departmentPath,
	}

	resp, err := s.GetUserWorkspacePermissions(ctx, req)
	if err != nil {
		return false, fmt.Errorf("获取权限记录失败: %w", err)
	}

	// ⭐ 使用响应对象的辅助方法检查权限（自动处理权限继承）
	return resp.CheckPermission(resourcePath, action), nil
}

// AddPolicy 添加权限策略（已废弃，仅使用角色系统）
func (s *PermissionServiceImpl) AddPolicy(ctx context.Context, username string, resourcePath string, action string) error {
	return fmt.Errorf("已废弃：请使用角色系统分配权限，不再支持直接权限点授权")
}

// RemovePolicy 删除权限策略（已废弃，仅使用角色系统）
func (s *PermissionServiceImpl) RemovePolicy(ctx context.Context, username string, resourcePath string, action string) error {
	return fmt.Errorf("已废弃：请使用角色系统管理权限，不再支持直接权限点删除")
}

// AddGroupingPolicy 添加关系策略（用户-角色关系，实现 enterprise.PermissionService 接口）
// ⭐ 新权限系统不支持角色，此方法已废弃
func (s *PermissionServiceImpl) AddGroupingPolicy(ctx context.Context, user string, role string) error {
	return fmt.Errorf("新权限系统不支持角色，请使用直接权限授权")
}

// RemoveGroupingPolicy 删除关系策略（用户-角色关系，实现 enterprise.PermissionService 接口）
// ⭐ 新权限系统不支持角色，此方法已废弃
func (s *PermissionServiceImpl) RemoveGroupingPolicy(ctx context.Context, user string, role string) error {
	return fmt.Errorf("新权限系统不支持角色，请使用直接权限授权")
}

// GetRolesForUser 获取用户的所有角色（实现 enterprise.PermissionService 接口）
// ⭐ 新权限系统不支持角色，此方法已废弃
func (s *PermissionServiceImpl) GetRolesForUser(ctx context.Context, username string) ([]string, error) {
	return []string{}, nil // 返回空列表
}

// AddResourceInheritance 添加资源继承关系（g2 关系，实现 enterprise.PermissionService 接口）
// ⭐ 新权限系统通过通配符路径自动实现继承，此方法已废弃
func (s *PermissionServiceImpl) AddResourceInheritance(ctx context.Context, childResource string, parentResource string) error {
	// 新权限系统通过通配符路径（如 /parent/*）自动实现继承
	// 不需要单独添加继承关系
	logger.Debugf(ctx, "[PermissionService] 新权限系统通过通配符路径自动实现继承，无需单独添加: child=%s, parent=%s",
		childResource, parentResource)
	return nil
}

// CreatePermissionRequest 创建权限申请（实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) CreatePermissionRequest(ctx context.Context, req *dto.CreatePermissionRequestReq) (int64, error) {
	// ⭐ 从 resourcePath 解析 user 和 app
	_, user, app := permissionpkg.ParseFullCodePath(req.ResourcePath)
	if user == "" || app == "" {
		return 0, fmt.Errorf("无法从资源路径解析 user 和 app: %s", req.ResourcePath)
	}

	// ⭐ 如果 AppID 为 0，尝试从 user 和 app 查询 AppID
	appID := req.AppID
	if appID == 0 && s.appRepo != nil {
		appModel, err := s.appRepo.GetAppByUserName(user, app)
		if err == nil && appModel != nil {
			appID = appModel.ID
		} else {
			logger.Warnf(ctx, "[PermissionService] 无法查询 AppID: user=%s, app=%s, error=%v", user, app, err)
		}
	}

	// 获取申请人用户名（从 context 或请求中获取）
	applicantUsername := req.ApplicantUsername
	if applicantUsername == "" {
		applicantUsername = contextx.GetRequestUser(ctx)
		if applicantUsername == "" {
			return 0, fmt.Errorf("无法获取申请人用户名")
		}
	}

	// 调用审批服务创建申请
	// ⭐ 直接使用 models.Time，无需转换
	approvalReq := &dto.InternalCreatePermissionRequestReq{
		User:              user,
		App:               app,
		AppID:             appID, // ⭐ 传递 AppID
		ApplicantUsername: applicantUsername,
		SubjectType:       req.SubjectType,
		Subject:           req.Subject,
		ResourcePath:      req.ResourcePath,
		RoleID:            req.RoleID, // ⭐ 角色ID（必填）
		StartTime:         req.StartTime, // ⭐ 直接赋值，无需转换
		EndTime:           req.EndTime,   // ⭐ 直接赋值，无需转换
		Reason:            req.Reason,
	}

	// ⭐ 调用内部审批服务
	request, err := s.approvalService.CreateRequest(ctx, approvalReq)
	if err != nil {
		return 0, fmt.Errorf("创建权限申请失败: %w", err)
	}

	return request.ID, nil
}

// CreateApprovalRequest 创建权限申请（审批流程，实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) CreateApprovalRequest(ctx context.Context, req *dto.InternalCreatePermissionRequestReq) (*dto.PermissionRequest, error) {
	return s.approvalService.CreateRequest(ctx, req)
}

// ApproveApprovalRequest 审批通过权限申请（实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) ApproveApprovalRequest(ctx context.Context, requestID int64, approverUsername string) error {
	return s.approvalService.ApproveRequest(ctx, requestID, approverUsername)
}

// RejectApprovalRequest 审批拒绝权限申请（实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) RejectApprovalRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error {
	return s.approvalService.RejectRequest(ctx, requestID, approverUsername, reason)
}

// ApprovePermissionRequest 审批通过权限申请（实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) ApprovePermissionRequest(ctx context.Context, requestID int64, approverUsername string) error {
	return s.ApproveApprovalRequest(ctx, requestID, approverUsername)
}

// RejectPermissionRequest 审批拒绝权限申请（实现 enterprise.PermissionService 接口）
func (s *PermissionServiceImpl) RejectPermissionRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error {
	return s.RejectApprovalRequest(ctx, requestID, approverUsername, reason)
}


// GetUserWorkspacePermissions 获取用户工作空间权限（服务树场景，实现 enterprise.PermissionService 接口）
// ⭐ 仅使用角色权限系统，不再查询 workspace_permission 表
func (s *PermissionServiceImpl) GetUserWorkspacePermissions(ctx context.Context, req *enterprise.GetUserWorkspacePermissionsReq) (*enterprise.GetUserWorkspacePermissionsResp, error) {
	// ⭐ 使用 user 和 app 查询（不再使用 appID）
	if req.User == "" || req.App == "" {
		return nil, fmt.Errorf("必须提供 user 和 app 参数")
	}

	// ⭐ 只查询用户角色权限（从内存缓存读取）
	userRolePermissions, err := s.permissionCalculator.getUserRolePermissions(ctx, req.User, req.App, req.Username, req.DepartmentPath)
	if err != nil {
		logger.Warnf(ctx, "[PermissionService] 查询用户角色权限失败: %v", err)
		// 返回空权限列表
		return &enterprise.GetUserWorkspacePermissionsResp{
			Records: []enterprise.PermissionRecord{},
		}, nil
	}

	// 构建权限记录列表
	records := make([]enterprise.PermissionRecord, 0)
	for resourcePath, actions := range userRolePermissions {
		for action := range actions {
			records = append(records, enterprise.PermissionRecord{
				Resource: resourcePath,
				Action:   action,
				Granted:  true,
				Perms:    actions,
			})
		}
	}

	return &enterprise.GetUserWorkspacePermissionsResp{
		Records: records,
	}, nil
}

// getResourceTypeFromServiceTree 从 ServiceTree 查询资源类型
// ⭐ 不能从路径推断，因为层级可能很深，无法判断是目录还是函数
func (s *PermissionServiceImpl) getResourceTypeFromServiceTree(ctx context.Context, resourcePath string) string {
	// 尝试从 ServiceTree 查询
	node, err := s.serviceTreeRepo.GetNodeByPath(ctx, resourcePath)
	if err == nil && node != nil {
		// 根据 ServiceTree.Type 转换为资源类型
		if node.Type == model.ServiceTreeTypeFunction {
			return "function"
		} else if node.Type == model.ServiceTreeTypePackage {
			return "directory"
		}
	}

	// 如果查询不到，检查是否是应用级别路径（/user/app）
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) == 2 {
		return "app"
	}

	// 默认返回 directory（因为大部分是目录）
	return "directory"
}
