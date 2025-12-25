package casbin

import (
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// CasbinService Casbin 权限引擎服务
// 封装 Casbin 的核心功能，提供权限检查、策略管理等
type CasbinService struct {
	enforcer *casbin.Enforcer
	db       *gorm.DB
}

// NewCasbinService 创建 Casbin 服务
// 参数：
//   - db: 数据库连接（用于存储权限策略）
//
// 返回：
//   - *CasbinService: Casbin 服务实例
//   - error: 如果初始化失败返回错误
//
// 说明：
//   - 使用 GORM 适配器，将权限策略存储在数据库中
//   - 启用自动保存和缓存，提升性能
func NewCasbinService(db *gorm.DB) (*CasbinService, error) {
	// 使用 GORM 适配器，将权限策略存储在数据库中
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// 加载模型文件
	enforcer, err := casbin.NewEnforcer("configs/casbin_model.conf", adapter)
	if err != nil {
		return nil, err
	}

	// ⭐ 显式加载策略（这会触发适配器创建 casbin_rule 表）
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("加载权限策略失败: %w", err)
	}

	// 启用自动保存（修改策略时自动保存到数据库）
	enforcer.EnableAutoSave(true)

	// 启用缓存（提升性能）
	enforcer.EnableCache(true)

	return &CasbinService{
		enforcer: enforcer,
		db:       db,
	}, nil
}

// CheckPermission 检查权限
// 参数：
//   - subject: 主体（用户或角色）
//   - object: 资源（资源路径）
//   - action: 操作（read、create、update、delete、execute）
//
// 返回：
//   - bool: 是否有权限
//   - error: 如果检查失败返回错误
func (s *CasbinService) CheckPermission(subject, object, action string) (bool, error) {
	return s.enforcer.Enforce(subject, object, action)
}

// BatchCheckPermissions 批量检查权限（优化性能）
// 参数：
//   - requests: 权限检查请求列表，每个请求格式为 [subject, object, action]
//
// 返回：
//   - []bool: 权限检查结果列表
//   - error: 如果检查失败返回错误
//
// 说明：
//   - 批量查询性能优秀，适合服务树权限标识等场景
func (s *CasbinService) BatchCheckPermissions(requests [][]interface{}) ([]bool, error) {
	return s.enforcer.BatchEnforce(requests)
}

// AddPolicy 添加策略
// 参数：
//   - params: 策略参数，格式为 [subject, object, action]
//
// 返回：
//   - bool: 是否添加成功
//   - error: 如果添加失败返回错误
func (s *CasbinService) AddPolicy(params ...string) (bool, error) {
	return s.enforcer.AddPolicy(params...)
}

// RemovePolicy 删除策略
// 参数：
//   - params: 策略参数，格式为 [subject, object, action]
//
// 返回：
//   - bool: 是否删除成功
//   - error: 如果删除失败返回错误
func (s *CasbinService) RemovePolicy(params ...string) (bool, error) {
	return s.enforcer.RemovePolicy(params...)
}

// AddGroupingPolicy 添加关系（g、g2、g3、g4）
// 参数：
//   - params: 关系参数，格式根据关系类型而定
//     - g: [user, role] - 用户-角色关系
//     - g2: [childResource, parentResource] - 资源继承关系
//     - g3: [user, department] - 用户-部门关系
//     - g4: [department, role] - 部门-角色关系
//
// 返回：
//   - bool: 是否添加成功
//   - error: 如果添加失败返回错误
func (s *CasbinService) AddGroupingPolicy(params ...string) (bool, error) {
	return s.enforcer.AddGroupingPolicy(params...)
}

// RemoveGroupingPolicy 删除关系
// 参数：
//   - params: 关系参数，格式根据关系类型而定
//
// 返回：
//   - bool: 是否删除成功
//   - error: 如果删除失败返回错误
func (s *CasbinService) RemoveGroupingPolicy(params ...string) (bool, error) {
	return s.enforcer.RemoveGroupingPolicy(params...)
}

// GetRolesForUser 获取用户的所有角色
// 参数：
//   - user: 用户名
//
// 返回：
//   - []string: 角色列表
//   - error: 如果查询失败返回错误
func (s *CasbinService) GetRolesForUser(user string) ([]string, error) {
	return s.enforcer.GetRolesForUser(user)
}

// GetRolesForUserWithType 获取用户的所有角色（指定关系类型）
// 参数：
//   - ptype: 关系类型（g、g2、g3、g4）
//   - user: 用户名
//
// 返回：
//   - []string: 角色列表
//   - error: 如果查询失败返回错误
//
// 说明：
//   - 用于获取用户-部门关系（g3）等特定类型的关系
func (s *CasbinService) GetRolesForUserWithType(ptype string, user string) ([]string, error) {
	return s.enforcer.GetRolesForUser(user)
}

// ReloadPolicy 重新加载策略（修改策略后调用）
// 返回：
//   - error: 如果重新加载失败返回错误
//
// 说明：
//   - 修改策略后，需要重新加载才能生效
func (s *CasbinService) ReloadPolicy() error {
	return s.enforcer.LoadPolicy()
}

// GetAllPolicies 获取所有策略（用于调试）
// 返回：
//   - [][]string: 所有策略列表，每个策略格式为 [subject, object, action]
//   - error: 如果查询失败返回错误
func (s *CasbinService) GetAllPolicies() ([][]string, error) {
	return s.enforcer.GetPolicy()
}

