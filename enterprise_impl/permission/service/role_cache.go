package service

import (
	"context"
	"sync"
	"time"

	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// RoleCache 角色缓存（全局单例）
type RoleCache struct {
	mu sync.RWMutex

	// 角色信息缓存：role_id -> Role
	roles map[int64]*Role

	// 角色权限缓存：role_id -> resourceType -> Set<action>
	rolePermissions map[int64]map[string]map[string]bool

	// 角色编码索引：resourceType:code -> role_id（支持同一 code 在不同资源类型中重复）
	roleCodeIndex map[string]int64

	// 最后更新时间
	lastUpdateTime time.Time

	// 缓存过期时间（默认 5 分钟）
	expireDuration time.Duration

	// 仓储依赖
	roleRepo            *permissionrepo.RoleRepository
	rolePermissionRepo  *permissionrepo.RolePermissionRepository
}

// Role 角色信息（缓存用）
type Role struct {
	ID          int64
	Name        string
	Code        string
	ResourceType string // ⭐ 资源类型
	Description string
	IsSystem    bool
	IsDefault   bool   // ⭐ 是否默认角色（用于权限申请时的默认推荐）
	CreatedBy   string // ⭐ 创建者
	CreatedAt   models.Time // ⭐ 创建时间
	UpdatedAt   models.Time // ⭐ 更新时间
	Permissions []string // 权限点列表（从 rolePermissions 获取）
}

// NewRoleCache 创建角色缓存
func NewRoleCache(roleRepo *permissionrepo.RoleRepository, rolePermissionRepo *permissionrepo.RolePermissionRepository) *RoleCache {
	return &RoleCache{
		roles:              make(map[int64]*Role),
		rolePermissions:    make(map[int64]map[string]map[string]bool),
		roleCodeIndex:      make(map[string]int64),
		expireDuration:     5 * time.Minute,
		roleRepo:            roleRepo,
		rolePermissionRepo:  rolePermissionRepo,
	}
}

// LoadAllRoles 加载所有角色到内存
func (c *RoleCache) LoadAllRoles(ctx context.Context) error {
	// 1. 查询所有角色
	roles, err := c.roleRepo.GetAllRoles(ctx)
	if err != nil {
		return err
	}

	// 2. 查询所有角色权限
	rolePerms, err := c.rolePermissionRepo.GetAllRolePermissions(ctx)
	if err != nil {
		return err
	}

	// 3. 构建缓存
	c.mu.Lock()
	defer c.mu.Unlock()

	c.roles = make(map[int64]*Role)
	c.rolePermissions = make(map[int64]map[string]map[string]bool)
	c.roleCodeIndex = make(map[string]int64)

	for _, role := range roles {
		c.roles[role.ID] = &Role{
			ID:           role.ID,
			Name:         role.Name,
			Code:         role.Code,
			ResourceType: role.ResourceType, // ⭐ 保存资源类型
			Description:  role.Description,
			IsSystem:     role.IsSystem,
			IsDefault:    role.IsDefault, // ⭐ 默认角色标记
			CreatedBy:    role.CreatedBy, // ⭐ 创建者
			CreatedAt:    role.CreatedAt, // ⭐ 创建时间
			UpdatedAt:    role.UpdatedAt, // ⭐ 更新时间
		}
		// ⭐ 使用 resourceType:code 作为索引，支持同一 code 在不同资源类型中重复
		indexKey := role.ResourceType + ":" + role.Code
		c.roleCodeIndex[indexKey] = role.ID
	}

	for _, rp := range rolePerms {
		// ⭐ 通过 ActionID 关联获取权限点信息
		if rp.ActionModel == nil {
			logger.Warnf(ctx, "[RoleCache] 角色权限的 Action 关联未加载: role_id=%d, action_id=%d", rp.RoleID, rp.ActionID)
			continue
		}
		
		actionCode := rp.ActionModel.Code
		resourceType := rp.ActionModel.ResourceType
		
		if c.rolePermissions[rp.RoleID] == nil {
			c.rolePermissions[rp.RoleID] = make(map[string]map[string]bool)
		}
		if c.rolePermissions[rp.RoleID][resourceType] == nil {
			c.rolePermissions[rp.RoleID][resourceType] = make(map[string]bool)
		}
		c.rolePermissions[rp.RoleID][resourceType][actionCode] = true
	}

	// 4. 更新最后更新时间
	c.lastUpdateTime = time.Now()

	logger.Infof(ctx, "[RoleCache] 加载角色缓存成功: 角色数=%d, 权限记录数=%d", len(c.roles), len(rolePerms))

	return nil
}

// Refresh 刷新缓存（角色变更时调用）
func (c *RoleCache) Refresh(ctx context.Context) error {
	return c.LoadAllRoles(ctx)
}

// GetRolePermissions 获取角色权限（从缓存读取，返回所有资源类型的权限）
func (c *RoleCache) GetRolePermissions(roleID int64) map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]bool)
	if permsByResource, ok := c.rolePermissions[roleID]; ok {
		// 合并所有资源类型的权限
		for _, perms := range permsByResource {
			for action := range perms {
				result[action] = true
			}
		}
	}

	return result
}

// GetRolePermissionsByResourceType 根据资源类型获取角色权限（从缓存读取）
func (c *RoleCache) GetRolePermissionsByResourceType(roleID int64, resourceType string) map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if permsByResource, ok := c.rolePermissions[roleID]; ok {
		if perms, ok := permsByResource[resourceType]; ok {
			// 返回副本，避免外部修改
			result := make(map[string]bool)
			for k, v := range perms {
				result[k] = v
			}
			return result
		}
	}

	return make(map[string]bool)
}

// GetRolePermissionsGroupedByResourceType 获取角色权限（按资源类型分组，从缓存读取）
func (c *RoleCache) GetRolePermissionsGroupedByResourceType(roleID int64) map[string]map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if permsByResource, ok := c.rolePermissions[roleID]; ok {
		// 返回副本，避免外部修改
		result := make(map[string]map[string]bool)
		for resourceType, perms := range permsByResource {
			result[resourceType] = make(map[string]bool)
			for action, granted := range perms {
				result[resourceType][action] = granted
			}
		}
		return result
	}

	return make(map[string]map[string]bool)
}

// GetRoleIDByCode 根据编码和资源类型获取角色ID
// ⭐ 支持同一 code 在不同资源类型中重复（如 chart:viewer 和 form:viewer）
func (c *RoleCache) GetRoleIDByCode(code string, resourceType string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	indexKey := resourceType + ":" + code
	roleID, ok := c.roleCodeIndex[indexKey]
	return roleID, ok
}

// GetRole 获取角色信息
func (c *RoleCache) GetRole(roleID int64) (*Role, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	role, ok := c.roles[roleID]
	if !ok {
		return nil, false
	}

	// 构建完整角色信息（包含权限）
	fullRole := &Role{
		ID:           role.ID,
		Name:         role.Name,
		Code:         role.Code,
		ResourceType: role.ResourceType, // ⭐ 保存资源类型
		Description:  role.Description,
		IsSystem:     role.IsSystem,
	}

	// 添加权限列表（合并所有资源类型的权限）
	if permsByResource, ok := c.rolePermissions[roleID]; ok {
		fullRole.Permissions = make([]string, 0)
		for _, perms := range permsByResource {
			for action := range perms {
				fullRole.Permissions = append(fullRole.Permissions, action)
			}
		}
	}

	return fullRole, true
}

// GetAllRoles 获取所有角色（从缓存读取）
func (c *RoleCache) GetAllRoles() []*Role {
	c.mu.RLock()
	defer c.mu.RUnlock()

	roles := make([]*Role, 0, len(c.roles))
	for _, role := range c.roles {
		// 构建完整角色信息（包含权限）
		fullRole := &Role{
			ID:           role.ID,
			Name:         role.Name,
			Code:         role.Code,
			ResourceType: role.ResourceType, // ⭐ 保存资源类型
			Description:  role.Description,
			IsSystem:     role.IsSystem,
			IsDefault:    role.IsDefault, // ⭐ 默认角色标记
			CreatedBy:    role.CreatedBy, // ⭐ 创建者
			CreatedAt:    role.CreatedAt, // ⭐ 创建时间
			UpdatedAt:    role.UpdatedAt, // ⭐ 更新时间
		}

		// 添加权限列表（合并所有资源类型的权限）
		if permsByResource, ok := c.rolePermissions[role.ID]; ok {
			fullRole.Permissions = make([]string, 0)
			for _, perms := range permsByResource {
				for action := range perms {
					fullRole.Permissions = append(fullRole.Permissions, action)
				}
			}
		}

		roles = append(roles, fullRole)
	}

	return roles
}

// StartRefreshTimer 启动定时刷新（可选）
func (c *RoleCache) StartRefreshTimer(ctx context.Context) {
	ticker := time.NewTicker(c.expireDuration)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.LoadAllRoles(ctx); err != nil {
					logger.Errorf(ctx, "[RoleCache] 刷新缓存失败: %v", err)
				} else {
					logger.Debugf(ctx, "[RoleCache] 缓存刷新成功")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
