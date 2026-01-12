package enterprise

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
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
	// 权限管理方法（企业版功能）
	// ============================================

	// AddPolicy 添加权限策略
	// 参数：
	//   - ctx: 上下文
	//   - username: 用户名
	//   - resourcePath: 资源路径（full-code-path）
	//   - action: 操作类型（如 table:search、function:manage 等）
	//
	// 返回：
	//   - error: 如果添加失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限管理）
	//   - 企业版实现会添加权限策略到自研权限系统
	AddPolicy(ctx context.Context, username string, resourcePath string, action string) error

	// RemovePolicy 删除权限策略
	// 参数：
	//   - ctx: 上下文
	//   - username: 用户名
	//   - resourcePath: 资源路径（full-code-path）
	//   - action: 操作类型
	//
	// 返回：
	//   - error: 如果删除失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限管理）
	//   - 企业版实现会从自研权限系统删除权限策略
	RemovePolicy(ctx context.Context, username string, resourcePath string, action string) error

	// AddGroupingPolicy 添加关系策略（用户-角色关系）
	// 参数：
	//   - ctx: 上下文
	//   - user: 用户名
	//   - role: 角色名称
	//
	// 返回：
	//   - error: 如果添加失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限管理）
	//   - 企业版实现会添加用户-角色关系到自研权限系统
	AddGroupingPolicy(ctx context.Context, user string, role string) error

	// RemoveGroupingPolicy 删除关系策略（用户-角色关系）
	// 参数：
	//   - ctx: 上下文
	//   - user: 用户名
	//   - role: 角色名称
	//
	// 返回：
	//   - error: 如果删除失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限管理）
	//   - 企业版实现会从自研权限系统删除用户-角色关系
	RemoveGroupingPolicy(ctx context.Context, user string, role string) error

	// GetRolesForUser 获取用户的所有角色
	// 参数：
	//   - ctx: 上下文
	//   - username: 用户名
	//
	// 返回：
	//   - []string: 角色列表
	//   - error: 如果查询失败返回错误
	//
	// 说明：
	//   - 社区版实现返回空列表
	//   - 企业版实现会从自研权限系统查询用户的所有角色
	GetRolesForUser(ctx context.Context, username string) ([]string, error)

	// AddResourceInheritance 添加资源继承关系（g2 关系）
	// 参数：
	//   - ctx: 上下文
	//   - childResource: 子资源路径
	//   - parentResource: 父资源路径
	//
	// 返回：
	//   - error: 如果添加失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限管理）
	//   - 企业版实现会添加资源继承关系到自研权限系统
	//   - 资源继承关系用于实现资源权限继承：如果父资源有权限，子资源自动继承
	AddResourceInheritance(ctx context.Context, childResource string, parentResource string) error

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
	CreatePermissionRequest(ctx context.Context, req *PermissionRequestReq) (int64, error)

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

	// GrantPermission 授权权限（管理员主动授权）
	// 参数：
	//   - ctx: 上下文
	//   - req: 授权请求
	//
	// 返回：
	//   - error: 如果授权失败返回错误
	//
	// 说明：
	//   - 社区版实现返回错误（不支持权限授权）
	//   - 企业版实现会直接创建权限记录，不需要审批
	GrantPermission(ctx context.Context, req *GrantPermissionReq) error

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
	CreateApprovalRequest(ctx context.Context, req *InternalCreatePermissionRequestReq) (*PermissionRequest, error)

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
}

// InternalCreatePermissionRequestReq 内部创建权限申请请求（企业版内部使用）
// ⭐ 注意：这个结构体应该与 dto.InternalCreatePermissionRequestReq 保持一致
// 但由于 enterprise 包不能依赖 dto 包，所以这里重新定义
type InternalCreatePermissionRequestReq struct {
	User              string       // 租户用户名
	App               string       // 应用代码
	ApplicantUsername string       // 申请人用户名
	SubjectType       string       // 权限主体类型
	Subject           string       // 权限主体
	ResourcePath      string       // 资源路径
	Action            string       // 操作类型
	StartTime         models.Time  // 权限开始时间
	EndTime           *models.Time // 权限结束时间（nil 表示永久）
	Reason            string       // 申请原因
}

// PermissionRequest 权限申请记录（企业版内部使用）
type PermissionRequest struct {
	ID                int64  // 申请记录ID
	AppID             int64  // 工作空间ID
	ApplicantUsername string // 申请人用户名
	SubjectType       string // 权限主体类型
	Subject           string // 权限主体
	ResourcePath      string // 资源路径
	Action            string // 操作类型
	Status            string // 申请状态
}

// PermissionRequestReq 权限申请请求
type PermissionRequestReq struct {
	AppID             int64
	ApplicantUsername string
	SubjectType       string       // user 或 department
	Subject           string       // 用户名或组织架构路径
	ResourcePath      string
	Action            string
	StartTime         models.Time  // 权限开始时间
	EndTime           *models.Time // 权限结束时间（nil 表示永久）
	Reason            string
}

// GrantPermissionReq 授权请求
type GrantPermissionReq struct {
	AppID           int64
	GrantorUsername string       // 授权人用户名
	GranteeType     string       // user 或 department
	Grantee         string       // 被授权人
	ResourcePath    string
	Action          string
	StartTime       models.Time  // 权限开始时间
	EndTime         *models.Time // 权限结束时间（nil 表示永久）
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
// ⭐ 支持权限继承：自动向上查找父节点的权限
// 例如检查 /user/app/l1/l2/l3 的 read 权限时，会检查：
//   - /user/app/l1/l2/l3 的 read 或 manage 权限
//   - /user/app/l1/l2 的 read 或 manage 权限
//   - /user/app/l1 的 read 或 manage 权限
//   - /user/app 的 read 或 manage 权限
func (r *GetUserWorkspacePermissionsResp) CheckPermission(resourcePath string, action string) bool {
	// 构建所有需要检查的路径（当前资源 + 父目录 + 应用级别）
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

	// 按优先级检查：直接权限 > 父目录权限 > 应用级别权限
	// ⭐ 目录权限自动继承：目录路径自动匹配所有子路径（如 /user/app/dir 匹配 /user/app/dir/function）
	for _, checkPath := range checkPaths {
		// 遍历所有权限记录，检查是否有匹配的目录权限
		for recordPath, actions := range permissionMap {
			// 1. 精确路径匹配
			if recordPath == checkPath {
				if actions[action] {
					return true
				}
				// 检查继承权限（manage 权限包含所有操作）
				if actions["directory:manage"] || actions["app:manage"] || actions["function:manage"] {
					return true
				}
			}

			// 2. 目录权限继承匹配：如果权限记录是目录路径，且 checkPath 是该目录的子路径，则匹配
			// 例如：权限记录是 /user/app/dir，checkPath 是 /user/app/dir/function，则匹配
			if len(recordPath) < len(checkPath) && strings.HasPrefix(checkPath, recordPath+"/") {
				// 检查直接权限
				if actions[action] {
					return true
				}
				// 检查权限继承：directory:read → function:read, directory:write → function:write 等
				if checkPermissionInheritance(action, actions) {
					return true
				}
				// 检查继承权限（manage 权限包含所有操作）
				if actions["directory:manage"] || actions["app:manage"] || actions["function:manage"] {
					return true
				}
			}
		}
	}

	return false
}

// checkPermissionInheritance 检查权限继承
// ⭐ 目录权限自动继承到子资源：
//   - directory:read → function:read
//   - directory:write → function:write
//   - directory:update → function:update
//   - directory:delete → function:delete
//   - app:manage → 所有权限
func checkPermissionInheritance(action string, actions map[string]bool) bool {
	// directory:read → function:read
	if action == "function:read" && actions["directory:read"] {
		return true
	}
	// directory:write → function:write
	if action == "function:write" && actions["directory:write"] {
		return true
	}
	// directory:update → function:update
	if action == "function:update" && actions["directory:update"] {
		return true
	}
	// directory:delete → function:delete
	if action == "function:delete" && actions["directory:delete"] {
		return true
	}
	// app:manage → 所有权限
	if actions["app:manage"] {
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

// AddPolicy 添加权限策略
// 社区版实现：返回错误（不支持权限管理）
func (u *UnImplPermissionService) AddPolicy(ctx context.Context, username string, resourcePath string, action string) error {
	return fmt.Errorf("权限管理功能仅在企业版可用")
}

// RemovePolicy 删除权限策略
// 社区版实现：返回错误（不支持权限管理）
func (u *UnImplPermissionService) RemovePolicy(ctx context.Context, username string, resourcePath string, action string) error {
	return fmt.Errorf("权限管理功能仅在企业版可用")
}

// AddGroupingPolicy 添加关系策略（用户-角色关系）
// 社区版实现：返回错误（不支持权限管理）
func (u *UnImplPermissionService) AddGroupingPolicy(ctx context.Context, user string, role string) error {
	return fmt.Errorf("权限管理功能仅在企业版可用")
}

// RemoveGroupingPolicy 删除关系策略（用户-角色关系）
// 社区版实现：返回错误（不支持权限管理）
func (u *UnImplPermissionService) RemoveGroupingPolicy(ctx context.Context, user string, role string) error {
	return fmt.Errorf("权限管理功能仅在企业版可用")
}

// GetRolesForUser 获取用户的所有角色
// 社区版实现：返回空列表
func (u *UnImplPermissionService) GetRolesForUser(ctx context.Context, username string) ([]string, error) {
	// 社区版（开源版本）默认实现：返回空列表
	return []string{}, nil
}

// AddResourceInheritance 添加资源继承关系（g2 关系）
// 社区版实现：返回错误（不支持权限管理）
func (u *UnImplPermissionService) AddResourceInheritance(ctx context.Context, childResource string, parentResource string) error {
	return fmt.Errorf("权限管理功能仅在企业版可用")
}

// CreatePermissionRequest 创建权限申请
// 社区版实现：返回错误（不支持权限申请）
func (u *UnImplPermissionService) CreatePermissionRequest(ctx context.Context, req *PermissionRequestReq) (int64, error) {
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

// GrantPermission 授权权限（管理员主动授权）
// 社区版实现：返回错误（不支持权限授权）
func (u *UnImplPermissionService) GrantPermission(ctx context.Context, req *GrantPermissionReq) error {
	return fmt.Errorf("权限授权功能仅在企业版可用")
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
func (u *UnImplPermissionService) CreateApprovalRequest(ctx context.Context, req *InternalCreatePermissionRequestReq) (*PermissionRequest, error) {
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
