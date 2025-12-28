package enterprise

import (
	"context"
	"fmt"
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

	// BatchCheckPermissions 批量检查权限
	// 用于服务树权限标识，批量查询多个资源的权限
	//
	// 参数：
	//   - ctx: 上下文
	//   - username: 用户名
	//   - resourcePaths: 资源路径列表
	//   - actions: 操作类型列表
	//
	// 返回：
	//   - map[string]map[string]bool: 权限结果（resourcePath -> action -> hasPermission）
	//   - error: 如果检查失败返回错误
	//
	// 说明：
	//   - 社区版实现返回所有权限为 true
	//   - 企业版实现会进行完整的批量权限检查
	BatchCheckPermissions(ctx context.Context, username string, resourcePaths []string, actions []string) (map[string]map[string]bool, error)

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
	//   - 企业版实现会添加权限策略到 Casbin
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
	//   - 企业版实现会从 Casbin 删除权限策略
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
	//   - 企业版实现会添加用户-角色关系到 Casbin
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
	//   - 企业版实现会从 Casbin 删除用户-角色关系
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
	//   - 企业版实现会从 Casbin 查询用户的所有角色
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
	//   - 企业版实现会添加 g2 关系到 Casbin
	//   - g2 关系用于实现资源权限继承：如果父资源有权限，子资源自动继承
	AddResourceInheritance(ctx context.Context, childResource string, parentResource string) error
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

// BatchCheckPermissions 批量检查权限
// 社区版实现：返回所有权限为 true
// 企业版实现：完整的批量权限检查
func (u *UnImplPermissionService) BatchCheckPermissions(ctx context.Context, username string, resourcePaths []string, actions []string) (map[string]map[string]bool, error) {
	// 社区版（开源版本）默认实现：返回所有权限为 true
	permissions := make(map[string]map[string]bool)
	for _, resourcePath := range resourcePaths {
		permissions[resourcePath] = make(map[string]bool)
		for _, action := range actions {
			permissions[resourcePath][action] = true
		}
	}
	return permissions, nil
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

