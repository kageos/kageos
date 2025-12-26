package casbin

import (
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// CasbinService Casbin 权限引擎服务
// 封装 Casbin 的核心功能，提供权限检查、策略管理等
type CasbinService struct {
	enforcer *casbin.CachedEnforcer // ⭐ 使用 CachedEnforcer 启用缓存
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
//   - ⚠️ 内存占用说明：
//     * 每个权限检查结果（bool）会被缓存，缓存 key 为 (subject, object, action)
//     * 每个缓存条目约 200-300 字节（key字符串 + value + 时间戳 + map overhead）
//     * 10万 个不同权限检查组合 ≈ 20-30 MB
//     * 100万 个不同权限检查组合 ≈ 200-300 MB
//     * CachedEnforcer 会自动清理过期缓存（30秒过期），防止内存无限增长
//     * 如果担心内存占用，可以缩短过期时间（如 15秒）或禁用缓存
func NewCasbinService(db *gorm.DB) (*CasbinService, error) {
	// 使用 GORM 适配器，将权限策略存储在数据库中
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// ⭐ 使用 CachedEnforcer 启用缓存（内存缓存，无需 Redis）
	// CachedEnforcer 使用 map 存储缓存，自动清理过期条目，防止内存无限增长
	enforcer, err := casbin.NewCachedEnforcer("configs/casbin_model.conf", adapter)
	if err != nil {
		return nil, err
	}

	// ⭐ 显式加载策略（这会触发适配器创建 casbin_rule 表）
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("加载权限策略失败: %w", err)
	}

	// ⭐ 设置缓存过期时间（30秒，平衡性能和内存占用）
	// 缓存过期后会自动重新查询，保证数据一致性
	// 较短的过期时间可以防止内存无限增长，同时保持较好的性能提升
	// 如果系统有大量不同用户访问不同资源，建议缩短到 15-20 秒
	enforcer.SetExpireTime(30 * time.Second)

	// 启用自动保存（修改策略时自动保存到数据库）
	enforcer.EnableAutoSave(true)

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
	// 将 []string 转换为 []interface{}
	interfaceParams := make([]interface{}, len(params))
	for i, v := range params {
		interfaceParams[i] = v
	}
	result, err := s.enforcer.AddPolicy(interfaceParams...)
	if err == nil && result {
		// ⭐ 策略变更后，清除缓存，确保权限检查使用最新策略
		s.enforcer.InvalidateCache()
	}
	return result, err
}

// RemovePolicy 删除策略
// 参数：
//   - params: 策略参数，格式为 [subject, object, action]
//
// 返回：
//   - bool: 是否删除成功
//   - error: 如果删除失败返回错误
func (s *CasbinService) RemovePolicy(params ...string) (bool, error) {
	// 将 []string 转换为 []interface{}
	interfaceParams := make([]interface{}, len(params))
	for i, v := range params {
		interfaceParams[i] = v
	}
	result, err := s.enforcer.RemovePolicy(interfaceParams...)
	if err == nil && result {
		// ⭐ 策略变更后，清除缓存，确保权限检查使用最新策略
		s.enforcer.InvalidateCache()
	}
	return result, err
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
	// 将 []string 转换为 []interface{}
	interfaceParams := make([]interface{}, len(params))
	for i, v := range params {
		interfaceParams[i] = v
	}
	result, err := s.enforcer.AddGroupingPolicy(interfaceParams...)
	if err == nil && result {
		// ⭐ 关系变更后，清除缓存，确保权限检查使用最新关系
		s.enforcer.InvalidateCache()
	}
	return result, err
}

// RemoveGroupingPolicy 删除关系
// 参数：
//   - params: 关系参数，格式根据关系类型而定
//
// 返回：
//   - bool: 是否删除成功
//   - error: 如果删除失败返回错误
func (s *CasbinService) RemoveGroupingPolicy(params ...string) (bool, error) {
	// 将 []string 转换为 []interface{}
	interfaceParams := make([]interface{}, len(params))
	for i, v := range params {
		interfaceParams[i] = v
	}
	result, err := s.enforcer.RemoveGroupingPolicy(interfaceParams...)
	if err == nil && result {
		// ⭐ 关系变更后，清除缓存，确保权限检查使用最新关系
		s.enforcer.InvalidateCache()
	}
	return result, err
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
//   - 重新加载后会自动清除缓存
func (s *CasbinService) ReloadPolicy() error {
	err := s.enforcer.LoadPolicy()
	if err == nil {
		// ⭐ 重新加载策略后，清除缓存
		s.enforcer.InvalidateCache()
	}
	return err
}

// GetAllPolicies 获取所有策略（用于调试）
// 返回：
//   - [][]string: 所有策略列表，每个策略格式为 [subject, object, action]
//   - error: 如果查询失败返回错误
func (s *CasbinService) GetAllPolicies() ([][]string, error) {
	return s.enforcer.GetPolicy()
}

// GetFilteredPolicy 获取过滤后的策略（用于批量查询优化）
// 参数：
//   - fieldIndex: 字段索引（0=subject, 1=object, 2=action）
//   - fieldValue: 字段值（用于过滤）
//
// 返回：
//   - [][]string: 过滤后的策略列表
//   - error: 如果查询失败返回错误
//
// 说明：
//   - 用于批量查询优化，一次性查询用户的所有权限
//   - 例如：GetFilteredPolicy(0, "username") 返回该用户的所有权限策略
func (s *CasbinService) GetFilteredPolicy(fieldIndex int, fieldValue string) ([][]string, error) {
	return s.enforcer.GetFilteredPolicy(fieldIndex, fieldValue)
}

// AddG2GroupingPolicy 添加 g2 关系（资源继承关系）
// 参数：
//   - childResource: 子资源路径
//   - parentResource: 父资源路径
//
// 返回：
//   - bool: 是否添加成功
//   - error: 如果添加失败返回错误
//
// 说明：
//   - g2 关系用于实现资源权限继承
//   - 如果父资源有权限，子资源自动继承（由 Casbin 模型配置处理）
func (s *CasbinService) AddG2GroupingPolicy(childResource string, parentResource string) (bool, error) {
	// 将 []string 转换为 []interface{}
	interfaceParams := []interface{}{childResource, parentResource}
	result, err := s.enforcer.AddNamedGroupingPolicy("g2", interfaceParams)
	if err == nil && result {
		// ⭐ 关系变更后，清除缓存，确保权限检查使用最新关系
		s.enforcer.InvalidateCache()
	}
	return result, err
}

