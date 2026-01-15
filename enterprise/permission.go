package enterprise

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// PermissionService 权限服务接口
// 用于权限检查、权限管理等
//
// 设计说明：
//   - 社区版：使用 UnImplPermissionService 空实现，不做权限控制
//   - 企业版：使用 enterprise_impl 中的具体实现，完整的权限控制
//
// 使用场景：
//   - 在中间件中调用，检查用户权限
//   - 在服务树中调用，批量检查权限
//   - 支持层级权限继承、组织架构权限等
//
// 实现位置：
//   - 开源实现：UnImplPermissionService（空实现，社区版使用）
//   - 企业实现：enterprise_impl/permission（闭源，企业版使用）
type PermissionService interface {
	Init // 继承初始化接口，企业实现需要初始化数据库等资源

	// CheckPermission 检查用户权限
	// 支持层级权限继承（应用 → 目录 → 函数 → 操作）
	//
	// 参数：
	//   - ctx: 上下文
	//   - username: 用户名
	//   - resourcePath: 资源路径（full-code-path）
	//   - action: 操作类型（read、create、update、delete、execute）
	//
	// 返回：
	//   - bool: 是否有权限
	//   - error: 如果检查失败返回错误
	//
	// 说明：
	//   - 社区版实现直接返回 true，不做权限控制
	//   - 企业版实现会进行完整的权限检查
	CheckPermission(ctx context.Context, username string, resourcePath string, action string) (bool, error)

	// ============================================
	// 权限申请和审批方法（企业版功能）
	// ============================================

	// CreatePermissionRequest 创建权限申请
	// 参数：
	//   - ctx: 上下文
	//   - req: 权限申请请求
	//
	// 返回：
	//   - requestID: 申请记录ID
	//   - error: 如果创建失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限申请）
	//   - 企业版实现会创建申请记录，状态为 pending
	CreatePermissionRequest(ctx context.Context, req *dto.CreatePermissionRequestReq) (int64, error)

	// ApprovePermissionRequest 审批通过权限申请
	// 参数：
	//   - ctx: 上下文
	//   - requestID: 申请记录ID
	//   - approverUsername: 审批人用户名
	//
	// 返回：
	//   - error: 如果审批失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限审批）
	//   - 企业版实现会更新申请状态为 approved，并创建权限记录
	ApprovePermissionRequest(ctx context.Context, requestID int64, approverUsername string) error

	// RejectPermissionRequest 审批拒绝权限申请
	// 参数：
	//   - ctx: 上下文
	//   - requestID: 申请记录ID
	//   - approverUsername: 审批人用户名
	//   - reason: 拒绝原因
	//
	// 返回：
	//   - error: 如果拒绝失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限审批）
	//   - 企业版实现会更新申请状态为 rejected
	RejectPermissionRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error

	// GetUserWorkspacePermissions 获取用户工作空间权限（服务树场景）
	// ⭐ 用于获取用户在工作空间中的所有权限记录，然后在应用层校验权限
	//
	// 参数：
	//   - ctx: 上下文
	//   - req: 权限查询请求
	//
	// 返回：
	//   - resp: 权限查询响应（包含辅助方法 CheckPermission）
	//   - error: 如果查询失败返回错误
	//
	// 说明：
	//   - 社区版实现返回空权限（所有权限为 false）
	//   - 企业版实现会查询用户和组织架构的所有权限
	//   - 返回的响应对象包含 CheckPermission 方法，方便在应用层校验权限
	GetUserWorkspacePermissions(ctx context.Context, req *GetUserWorkspacePermissionsReq) (*GetUserWorkspacePermissionsResp, error)

	// ============================================
	// 审批服务方法（企业版功能，原 ApprovalService 的方法）
	// ============================================

	// CreateApprovalRequest 创建权限申请（审批流程）
	// 参数：
	//   - ctx: 上下文
	//   - req: 内部创建权限申请请求
	//
	// 返回：
	//   - request: 权限申请记录
	//   - error: 如果创建失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限申请）
	//   - 企业版实现会创建申请记录，状态为 pending
	CreateApprovalRequest(ctx context.Context, req *dto.InternalCreatePermissionRequestReq) (*dto.PermissionRequest, error)

	// ApproveApprovalRequest 审批通过权限申请
	// 参数：
	//   - ctx: 上下文
	//   - requestID: 申请记录ID
	//   - approverUsername: 审批人用户名
	//
	// 返回：
	//   - error: 如果审批失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限审批）
	//   - 企业版实现会更新申请状态为 approved，并创建权限记录
	ApproveApprovalRequest(ctx context.Context, requestID int64, approverUsername string) error

	// RejectApprovalRequest 审批拒绝权限申请
	// 参数：
	//   - ctx: 上下文
	//   - requestID: 申请记录ID
	//   - approverUsername: 审批人用户名
	//   - reason: 拒绝原因
	//
	// 返回：
	//   - error: 如果拒绝失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限审批）
	//   - 企业版实现会更新申请状态为 rejected
	RejectApprovalRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error

	// ============================================
	// 角色管理方法（企业版功能）
	// ============================================

	// GetRoles 获取所有角色
	// 参数：
	//   - ctx: 上下文
	//   - resourceType: 资源类型过滤（可选，为空则返回所有角色）
	//
	// 返回：
	//   - *dto.GetRolesResp: 角色列表响应
	//   - error: 如果获取失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会从内存缓存读取角色，如果指定了 resourceType 则只返回该资源类型的角色
	GetRoles(ctx context.Context, resourceType string) (*dto.GetRolesResp, error)

	// GetRole 获取角色详情
	// 参数：
	//   - ctx: 上下文
	//   - roleID: 角色ID
	//
	// 返回：
	//   - *dto.GetRoleResp: 角色响应
	//   - error: 如果获取失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会查询角色详情（包含权限列表）
	GetRole(ctx context.Context, roleID int64) (*dto.GetRoleResp, error)

	// CreateRole 创建角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 创建角色请求
	//
	// 返回：
	//   - *dto.CreateRoleResp: 创建角色响应
	//   - error: 如果创建失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会创建角色并配置权限
	CreateRole(ctx context.Context, req *dto.CreateRoleReq) (*dto.CreateRoleResp, error)

	// UpdateRole 更新角色
	// 参数：
	//   - ctx: 上下文
	//   - roleID: 角色ID
	//   - req: 更新角色请求
	//
	// 返回：
	//   - *dto.UpdateRoleResp: 更新角色响应
	//   - error: 如果更新失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会更新角色信息或权限配置
	UpdateRole(ctx context.Context, roleID int64, req *dto.UpdateRoleReq) (*dto.UpdateRoleResp, error)

	// DeleteRole 删除角色
	// 参数：
	//   - ctx: 上下文
	//   - roleID: 角色ID
	//
	// 返回：
	//   - error: 如果删除失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会删除自定义角色（系统角色不能删除）
	DeleteRole(ctx context.Context, roleID int64) error

	// AssignRoleToUser 给用户分配角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 分配角色请求
	//
	// 返回：
	//   - *dto.AssignRoleToUserResp: 分配角色响应
	//   - error: 如果分配失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会给指定用户分配角色，指定资源路径
	AssignRoleToUser(ctx context.Context, req *dto.AssignRoleToUserReq) (*dto.AssignRoleToUserResp, error)

	// AssignRoleToDepartment 给组织架构分配角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 分配角色请求
	//
	// 返回：
	//   - *dto.AssignRoleToDepartmentResp: 分配角色响应
	//   - error: 如果分配失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会给指定组织架构分配角色，组织架构下所有成员自动获得权限
	AssignRoleToDepartment(ctx context.Context, req *dto.AssignRoleToDepartmentReq) (*dto.AssignRoleToDepartmentResp, error)

	// RemoveRoleFromUser 移除用户角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 移除角色请求
	//
	// 返回：
	//   - error: 如果移除失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会移除用户的角色分配
	RemoveRoleFromUser(ctx context.Context, req *dto.RemoveRoleFromUserReq) error

	// RemoveRoleFromDepartment 移除组织架构角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 移除角色请求
	//
	// 返回：
	//   - error: 如果移除失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会移除组织架构的角色分配
	RemoveRoleFromDepartment(ctx context.Context, req *dto.RemoveRoleFromDepartmentReq) error

	// GetUserRoles 获取用户角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 获取用户角色请求
	//
	// 返回：
	//   - *dto.GetUserRolesResp: 用户角色响应
	//   - error: 如果获取失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会查询指定用户的所有角色分配
	GetUserRoles(ctx context.Context, req *dto.GetUserRolesReq) (*dto.GetUserRolesResp, error)

	// GetDepartmentRoles 获取组织架构角色
	// 参数：
	//   - ctx: 上下文
	//   - req: 获取组织架构角色请求
	//
	// 返回：
	//   - *dto.GetDepartmentRolesResp: 组织架构角色响应
	//   - error: 如果获取失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会查询指定组织架构的所有角色分配
	GetDepartmentRoles(ctx context.Context, req *dto.GetDepartmentRolesReq) (*dto.GetDepartmentRolesResp, error)

	// GetRolesForPermissionRequest 获取可用于权限申请的角色列表（根据节点类型过滤）
	// 参数：
	//   - ctx: 上下文
	//   - req: 获取角色列表请求
	//
	// 返回：
	//   - *dto.GetRolesForPermissionRequestResp: 角色列表响应
	//   - error: 如果获取失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持角色管理）
	//   - 企业版实现会根据节点类型和模板类型，返回包含该资源类型权限的角色列表
	GetRolesForPermissionRequest(ctx context.Context, req *dto.GetRolesForPermissionRequestReq) (*dto.GetRolesForPermissionRequestResp, error)
}

// GetUserWorkspacePermissionsReq 获取用户工作空间权限请求
type GetUserWorkspacePermissionsReq struct {
	User           string // 租户用户名
	App            string // 应用代码
	Username       string
	DepartmentPath string // 组织架构路径
}

// GetUserWorkspacePermissionsResp 获取用户工作空间权限响应
type GetUserWorkspacePermissionsResp struct {
	Records []PermissionRecord // 权限记录列表
}

// PermissionRecord 权限记录
type PermissionRecord struct {
	Resource string          // 资源路径
	Action   string          // 操作类型
	Granted  bool            // 是否有权限
	Perms    map[string]bool // 该资源的所有权限点
}

// CheckPermission 检查指定资源路径和操作是否有权限
// ⭐ 简化逻辑：直接检查父目录是否有相同的权限点（不需要转换）
// 例如检查 /user/app/dir/function 的 table:read 权限时：
//  1. 先检查 /user/app/dir/function 是否有 table:read 权限
//  2. 如果没有，检查父目录 /user/app/dir 是否有 table:read 权限（直接继承）
//  3. 如果还没有，继续向上检查父目录
func (r *GetUserWorkspacePermissionsResp) CheckPermission(resourcePath string, action string) bool {
	// 构建所有需要检查的路径（当前资源 + 所有父目录 + 应用级别）
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) < 2 {
		return false
	}

	// 构建检查路径列表（优先级从高到低）
	checkPaths := []string{resourcePath}

	// 添加所有父目录路径
	for i := len(parts) - 1; i >= 2; i-- {
		parentPath := "/" + strings.Join(parts[:i], "/")
		checkPaths = append(checkPaths, parentPath)
	}

	// 添加应用级别路径
	appPath := "/" + strings.Join(parts[:2], "/")
	checkPaths = append(checkPaths, appPath)

	// 构建权限映射（resourcePath -> Set<action>）
	permissionMap := make(map[string]map[string]bool)
	for _, record := range r.Records {
		if permissionMap[record.Resource] == nil {
			permissionMap[record.Resource] = make(map[string]bool)
		}
		permissionMap[record.Resource][record.Action] = true
	}

	// ⭐ 简化逻辑：按优先级检查每个路径，如果该路径有相同的权限点，就放行
	for _, checkPath := range checkPaths {
		if actions, ok := permissionMap[checkPath]; ok {
			// 1. 直接检查是否有相同的权限点（例如：table:read）
			if actions[action] {
				return true
			}

			// 2. 检查 admin 权限（包含所有操作）- 动态检查所有资源类型的 admin 权限
			// 遍历所有权限点，检查是否有任何资源类型的 admin 权限
			for actionKey := range actions {
				if strings.HasSuffix(actionKey, ":admin") {
					return true
				}
			}

			// 3. 检查权限继承（directory:read → table:read 等）
			if checkPermissionInheritance(action, actions) {
				return true
			}
		}
	}

	return false
}

// checkPermissionInheritance 检查权限继承
// ⭐ 目录权限自动继承到子资源：
//   - directory:read → table:read, form:read, chart:read
//   - directory:write → table:write, form:write
//   - directory:update → table:update
//   - directory:delete → table:delete
//   - 任何资源类型的 admin 权限 → 所有权限
func checkPermissionInheritance(action string, actions map[string]bool) bool {
	// ⭐ 检查是否有任何资源类型的 admin 权限（动态检查，不硬编码）
	for actionKey := range actions {
		if strings.HasSuffix(actionKey, ":admin") {
			return true
		}
	}

	// ⭐ 目录权限继承到子资源
	// directory:read → table:read, form:read, chart:read
	if (action == "table:read" || action == "form:read" || action == "chart:read") && actions["directory:read"] {
		return true
	}
	// directory:write → table:write, form:write
	if (action == "table:write" || action == "form:write") && actions["directory:write"] {
		return true
	}
	// directory:update → table:update
	if action == "table:update" && actions["directory:update"] {
		return true
	}
	// directory:delete → table:delete
	if action == "table:delete" && actions["directory:delete"] {
		return true
	}

	return false
}

// 全局变量：存储当前实现
var permissionServiceImpl PermissionService = &UnImplPermissionService{}

// RegisterPermissionService 注册权限服务实现
// 企业版在 init() 中调用此函数注册真实实现
func RegisterPermissionService(impl PermissionService) {
	permissionServiceImpl = impl
}

// GetPermissionService 获取当前权限服务
// 业务代码通过此函数获取实现（社区版或企业版）
func GetPermissionService() PermissionService {
	return permissionServiceImpl
}

// InitPermissionService 初始化权限服务
// 用于在系统启动时初始化权限功能
//
// 参数：
//   - opt: 初始化选项，包含数据库连接等依赖
//
// 返回：
//   - error: 如果初始化失败返回错误
//
// 说明：
//   - 自动使用已注册的实现（社区版或企业版）
//   - 企业版需要在 init() 中调用 RegisterPermissionService() 注册
func InitPermissionService(opt *InitOptions) error {
	return permissionServiceImpl.Init(opt)
}

// UnImplPermissionService 未实现的权限服务（社区版）
// 这是开源版本使用的实现，企业实现会替换为完整实现
//
// 设计目的：
//   - 保持接口一致性，社区版和企业版使用相同的接口
//   - 企业实现会替换为完整实现，提供完整的权限控制
//
// 策略说明：
//   - 社区版：不做权限控制，所有权限检查返回 true
//   - 企业版：完整的权限控制，支持层级权限继承、组织架构权限等
//
// 使用场景：
//   - 开源项目默认使用此实现（空实现，不做权限控制）
//   - 企业版用户购买许可证后，替换为企业实现（完整权限控制）
type UnImplPermissionService struct {
	// 空结构体，不需要任何字段
}

// Init 初始化方法（空实现）
// 社区版不需要初始化任何资源，直接返回成功
func (u *UnImplPermissionService) Init(opt *InitOptions) error {
	return nil
}

// CheckPermission 检查用户权限
// 社区版实现：直接返回 true，不做权限控制
// 企业版实现：完整的权限检查，支持层级权限继承
func (u *UnImplPermissionService) CheckPermission(ctx context.Context, username string, resourcePath string, action string) (bool, error) {
	// 社区版（开源版本）默认实现：不做权限控制，直接返回 true
	return true, nil
}

// CreatePermissionRequest 创建权限申请
// 社区版实现：返回错误（不支持权限申请）
func (u *UnImplPermissionService) CreatePermissionRequest(ctx context.Context, req *dto.CreatePermissionRequestReq) (int64, error) {
	return 0, fmt.Errorf("权限申请功能仅在企业版可用")
}

// ApprovePermissionRequest 审批通过权限申请
// 社区版实现：返回错误（不支持权限审批）
func (u *UnImplPermissionService) ApprovePermissionRequest(ctx context.Context, requestID int64, approverUsername string) error {
	return fmt.Errorf("权限审批功能仅在企业版可用")
}

// RejectPermissionRequest 审批拒绝权限申请
// 社区版实现：返回错误（不支持权限审批）
func (u *UnImplPermissionService) RejectPermissionRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error {
	return fmt.Errorf("权限审批功能仅在企业版可用")
}

// GetUserWorkspacePermissions 获取用户工作空间权限（服务树场景）
// 社区版实现：返回空权限
func (u *UnImplPermissionService) GetUserWorkspacePermissions(ctx context.Context, req *GetUserWorkspacePermissionsReq) (*GetUserWorkspacePermissionsResp, error) {
	return &GetUserWorkspacePermissionsResp{
		Records: []PermissionRecord{},
	}, nil
}

// CreateApprovalRequest 创建权限申请（审批流程）
// 社区版实现：返回错误（不支持权限申请）
func (u *UnImplPermissionService) CreateApprovalRequest(ctx context.Context, req *dto.InternalCreatePermissionRequestReq) (*dto.PermissionRequest, error) {
	return nil, fmt.Errorf("权限申请功能仅在企业版可用")
}

// ApproveApprovalRequest 审批通过权限申请
// 社区版实现：返回错误（不支持权限审批）
func (u *UnImplPermissionService) ApproveApprovalRequest(ctx context.Context, requestID int64, approverUsername string) error {
	return fmt.Errorf("权限审批功能仅在企业版可用")
}

// RejectApprovalRequest 审批拒绝权限申请
// 社区版实现：返回错误（不支持权限审批）
func (u *UnImplPermissionService) RejectApprovalRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error {
	return fmt.Errorf("权限审批功能仅在企业版可用")
}

// ============================================
// 角色管理方法（社区版空实现）
// ============================================

// GetRoles 获取所有角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) GetRoles(ctx context.Context, resourceType string) (*dto.GetRolesResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// GetRole 获取角色详情
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) GetRole(ctx context.Context, roleID int64) (*dto.GetRoleResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// CreateRole 创建角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) CreateRole(ctx context.Context, req *dto.CreateRoleReq) (*dto.CreateRoleResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// UpdateRole 更新角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) UpdateRole(ctx context.Context, roleID int64, req *dto.UpdateRoleReq) (*dto.UpdateRoleResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// DeleteRole 删除角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) DeleteRole(ctx context.Context, roleID int64) error {
	return fmt.Errorf("角色管理功能仅在企业版可用")
}

// AssignRoleToUser 给用户分配角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) AssignRoleToUser(ctx context.Context, req *dto.AssignRoleToUserReq) (*dto.AssignRoleToUserResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// AssignRoleToDepartment 给组织架构分配角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) AssignRoleToDepartment(ctx context.Context, req *dto.AssignRoleToDepartmentReq) (*dto.AssignRoleToDepartmentResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// RemoveRoleFromUser 移除用户角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) RemoveRoleFromUser(ctx context.Context, req *dto.RemoveRoleFromUserReq) error {
	return fmt.Errorf("角色管理功能仅在企业版可用")
}

// RemoveRoleFromDepartment 移除组织架构角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) RemoveRoleFromDepartment(ctx context.Context, req *dto.RemoveRoleFromDepartmentReq) error {
	return fmt.Errorf("角色管理功能仅在企业版可用")
}

// GetUserRoles 获取用户角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) GetUserRoles(ctx context.Context, req *dto.GetUserRolesReq) (*dto.GetUserRolesResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// GetDepartmentRoles 获取组织架构角色
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) GetDepartmentRoles(ctx context.Context, req *dto.GetDepartmentRolesReq) (*dto.GetDepartmentRolesResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}

// GetRolesForPermissionRequest 获取可用于权限申请的角色列表
// 社区版实现：返回错误（不支持角色管理）
func (u *UnImplPermissionService) GetRolesForPermissionRequest(ctx context.Context, req *dto.GetRolesForPermissionRequestReq) (*dto.GetRolesForPermissionRequestResp, error) {
	return nil, fmt.Errorf("角色管理功能仅在企业版可用")
}
