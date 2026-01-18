package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// ============================================
// 权限计算器 V2 - 权限判断逻辑说明
// ============================================
//
// 权限继承方式：方式2（直接函数权限继承）
//
// 工作原理：
//   - 如果父目录配置了 table:write，子函数直接继承 table:write 权限
//   - 如果父目录配置了 form:read，子函数直接继承 form:read 权限
//   - 如果父目录配置了 table:admin，子函数直接继承 table:admin 权限（拥有所有表格权限）
//   - 不需要转换，直接匹配相同的权限点
//
// 示例：
//   父目录：/user/app/dir
//   角色：目录开发者（配置了 table:write, form:read）
//
//   子函数：/user/app/dir/function1
//   检查 table:write 权限：
//     ✅ 父目录有 table:write → 子函数继承 table:write
//
//   父目录：/user/app/dir
//   角色：目录管理员（配置了 table:admin, form:admin, chart:admin）
//
//   子函数：/user/app/dir/function1
//   检查 table:read 权限：
//     ✅ 父目录有 table:admin → 子函数继承 table:admin（包含所有表格权限）
//
// 节点权限解析流程（NodePermissionResolver.Resolve）：
//   1. checkExactPath：检查精确路径匹配
//   2. checkParentPaths：检查父目录路径（权限继承）- 方式2
//   3. checkPrefixMatch：检查前缀匹配（支持深层嵌套）- 方式2
//   4. checkAppLevelPermissions：检查应用级别权限
//
// 已废弃的方式1（目录权限转换）：
//   - directory:read → table:read（已删除 applyDirectoryInheritance 方法）
//   - directory:write → table:write（已删除）
//   原因：方式1太混乱，只使用方式2（直接函数权限继承）
//
// ============================================

// ============================================
// 权限计算器接口（统一新旧版本）
// ============================================

// PermissionCalculatorInterface 权限计算器接口
type PermissionCalculatorInterface interface {
	// CalculateWorkspacePermissions 计算工作空间所有节点的权限
	CalculateWorkspacePermissions(
		ctx context.Context,
		user string,
		app string,
		username string,
		departmentPath string,
		trees []*model.ServiceTree,
	) (map[string]map[string]bool, error)

	// getUserRolePermissions 获取用户角色权限（统一查询）
	getUserRolePermissions(
		ctx context.Context,
		user string,
		app string,
		username string,
		departmentPath string,
	) (map[string]map[string]bool, error)
}

// ============================================
// 权限计算模块（结构化设计）
// ============================================

// RolePermissionMap 角色权限映射（按路径和资源类型组织）
type RolePermissionMap struct {
	// 数据结构：resourcePath -> resourceType -> action -> true
	data map[string]map[string]map[string]bool
}

// NewRolePermissionMap 创建角色权限映射
func NewRolePermissionMap() *RolePermissionMap {
	return &RolePermissionMap{
		data: make(map[string]map[string]map[string]bool),
	}
}

// Add 添加权限到映射
func (m *RolePermissionMap) Add(resourcePath string, resourceType string, action string) {
	if m.data[resourcePath] == nil {
		m.data[resourcePath] = make(map[string]map[string]bool)
	}
	if m.data[resourcePath][resourceType] == nil {
		m.data[resourcePath][resourceType] = make(map[string]bool)
	}
	m.data[resourcePath][resourceType][action] = true
}

// Get 获取指定路径和资源类型的权限
func (m *RolePermissionMap) Get(resourcePath string, resourceType string) map[string]bool {
	if m.data[resourcePath] == nil {
		return nil
	}
	return m.data[resourcePath][resourceType]
}

// HasPath 检查是否存在指定路径的权限
func (m *RolePermissionMap) HasPath(resourcePath string) bool {
	_, ok := m.data[resourcePath]
	return ok
}

// GetAllPaths 获取所有路径
func (m *RolePermissionMap) GetAllPaths() []string {
	paths := make([]string, 0, len(m.data))
	for path := range m.data {
		paths = append(paths, path)
	}
	return paths
}

// NodePermissionResolver 节点权限解析器（负责为单个节点计算权限）
type NodePermissionResolver struct {
	rolePermissionMap *RolePermissionMap
	resourceType      string
	requiredActions   []string
}

// NewNodePermissionResolver 创建节点权限解析器
func NewNodePermissionResolver(
	rolePermissionMap *RolePermissionMap,
	resourceType string,
	requiredActions []string,
) *NodePermissionResolver {
	return &NodePermissionResolver{
		rolePermissionMap: rolePermissionMap,
		resourceType:      resourceType,
		requiredActions:   requiredActions,
	}
}

// Resolve 解析节点权限
//
// ⭐ 权限解析流程（按优先级）：
//   1. checkExactPath：检查精确路径匹配
//      - 检查当前资源路径是否有该权限点
//      - 例如：检查 /user/app/dir/function 是否有 table:read 权限
//
//   2. checkParentPaths：检查父目录路径（权限继承 - 方式2：直接函数权限继承）
//      - 向上查找父目录，检查是否有相同的权限点（直接继承，不需要转换）
//      - 例如：父目录 /user/app/dir 配置了 table:write，子函数 /user/app/dir/function 直接继承 table:write
//      - 例如：父目录配置了 table:admin，子函数继承 table:admin（拥有所有表格权限）
//
//   3. checkPrefixMatch：检查前缀匹配（支持深层嵌套）
//      - 检查是否有前缀路径配置了该权限点
//      - 例如：/user/app/* 配置了 table:read，那么 /user/app/dir/function 继承该权限
//
//   4. checkAppLevelPermissions：检查应用级别权限
//      - 如果用户有 app:admin 权限，拥有该应用下所有资源的权限
//
// ⭐ 权限继承方式：方式2（直接函数权限继承）
//   - 不需要转换，直接匹配相同的权限点
//   - 例如：父目录配置了 table:write → 子函数直接继承 table:write
//   - 例如：父目录配置了 form:read → 子函数直接继承 form:read
//
func (r *NodePermissionResolver) Resolve(nodePath string) map[string]bool {
	// 初始化节点权限（默认全部为 false）
	nodePerms := make(map[string]bool)
	for _, action := range r.requiredActions {
		nodePerms[action] = false
	}

	// 1. 检查精确路径匹配
	r.checkExactPath(nodePath, nodePerms)

	// 2. 检查父目录路径（目录权限继承 - 方式2：直接函数权限继承）
	r.checkParentPaths(nodePath, nodePerms)

	// 3. 检查前缀匹配（支持深层嵌套）
	r.checkPrefixMatch(nodePath, nodePerms)

	// 4. 检查应用级别权限
	r.checkAppLevelPermissions(nodePath, nodePerms)

	return nodePerms
}

// checkExactPath 检查精确路径匹配
func (r *NodePermissionResolver) checkExactPath(nodePath string, nodePerms map[string]bool) {
	rolePerms := r.rolePermissionMap.Get(nodePath, r.resourceType)
	for actionCode := range rolePerms {
		for _, requiredAction := range r.requiredActions {
			if actionCode == requiredAction {
				nodePerms[requiredAction] = true
				break
			}
		}
	}
}

// checkParentPaths 检查父目录路径（权限继承）
// ⭐ 只支持直接函数权限继承：如果父目录配置了 table:write，子函数直接继承 table:write
// 例如：父目录 a/b 的开发者角色配置了 table:write，那么子函数 a/b/c 直接继承 table:write 权限
func (r *NodePermissionResolver) checkParentPaths(nodePath string, nodePerms map[string]bool) {
	parentPaths := permission.GetParentPaths(nodePath)
	for _, parentPath := range parentPaths {
		if !r.rolePermissionMap.HasPath(parentPath) {
			continue
		}

		// ⭐ 检查父目录直接配置的函数权限（直接继承）
		// 例如：父目录 a/b 的开发者角色配置了 table:write
		// 那么子函数 a/b/c 直接继承 table:write 权限
		rolePerms := r.rolePermissionMap.Get(parentPath, r.resourceType)
		if rolePerms != nil {
			for actionCode := range rolePerms {
				for _, requiredAction := range r.requiredActions {
					if actionCode == requiredAction {
						nodePerms[requiredAction] = true
						break
					}
				}
			}
		}
	}
}

// checkPrefixMatch 检查前缀匹配（支持深层嵌套）
func (r *NodePermissionResolver) checkPrefixMatch(nodePath string, nodePerms map[string]bool) {
	// 如果已经有权限了，不需要前缀匹配
	if r.hasAnyPermission(nodePerms) {
		return
	}

	// 遍历所有角色分配路径，检查是否是节点路径的前缀
	for assignedPath := range r.rolePermissionMap.data {
		if strings.HasPrefix(nodePath, assignedPath+"/") || nodePath == assignedPath {
			// ⭐ 检查父目录直接配置的函数权限（直接继承）
			// 例如：父目录 a/b 的开发者角色配置了 table:write
			// 那么子函数 a/b/c 直接继承 table:write 权限
			rolePerms := r.rolePermissionMap.Get(assignedPath, r.resourceType)
			if rolePerms != nil {
				for actionCode := range rolePerms {
					for _, requiredAction := range r.requiredActions {
						if actionCode == requiredAction {
							nodePerms[requiredAction] = true
							break
						}
					}
				}
			}
		}
	}
}

// checkAppLevelPermissions 检查应用级别权限
func (r *NodePermissionResolver) checkAppLevelPermissions(nodePath string, nodePerms map[string]bool) {
	appPath := getAppPath(nodePath)
	if appPath == "" {
		return
	}

	appPerms := r.rolePermissionMap.Get(appPath, permission.ResourceTypeApp)
	appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
	if appPerms[appAdminCode] {
		// app:admin -> 所有权限
		for _, requiredAction := range r.requiredActions {
			nodePerms[requiredAction] = true
		}
	}
}

// hasAnyPermission 检查是否有任何权限
func (r *NodePermissionResolver) hasAnyPermission(perms map[string]bool) bool {
	for _, hasPerm := range perms {
		if hasPerm {
			return true
		}
	}
	return false
}

// PermissionInheritanceApplier 权限继承应用器（负责处理节点间的权限继承）
type PermissionInheritanceApplier struct{}

// NewPermissionInheritanceApplier 创建权限继承应用器
func NewPermissionInheritanceApplier() *PermissionInheritanceApplier {
	return &PermissionInheritanceApplier{}
}

// Apply 应用权限继承规则
func (a *PermissionInheritanceApplier) Apply(
	nodeType string,
	templateType string,
	parentPerms map[string]bool,
	nodePerms map[string]bool,
) {
	// 1. admin → 子节点自动拥有所有权限
	if parentPerms[permission.ActionAdmin] {
		for action := range nodePerms {
			nodePerms[action] = true
		}
		return
	}

	// 2. read/write/update/delete → 直接继承（不需要区分资源类型）
	if parentPerms[permission.ActionRead] {
		nodePerms[permission.ActionRead] = true
	}
	if parentPerms[permission.ActionWrite] {
		nodePerms[permission.ActionWrite] = true
	}
	if parentPerms[permission.ActionUpdate] {
		nodePerms[permission.ActionUpdate] = true
	}
	if parentPerms[permission.ActionDelete] {
		nodePerms[permission.ActionDelete] = true
	}
}

// RoleAssignmentLoader 角色分配加载器（负责加载和构建角色权限映射）
type RoleAssignmentLoader struct {
	roleAssignmentRepo *repository.RoleAssignmentRepository
	roleCache           *RoleCache
}

// NewRoleAssignmentLoader 创建角色分配加载器
func NewRoleAssignmentLoader(
	roleAssignmentRepo *repository.RoleAssignmentRepository,
	roleCache *RoleCache,
) *RoleAssignmentLoader {
	return &RoleAssignmentLoader{
		roleAssignmentRepo: roleAssignmentRepo,
		roleCache:           roleCache,
	}
}

// Load 加载角色分配并构建权限映射
func (l *RoleAssignmentLoader) Load(
	ctx context.Context,
	user string,
	app string,
	subjects []permissionrepo.SubjectInfo,
) (*RolePermissionMap, error) {
	// 查询角色分配
	roleAssignments, err := l.roleAssignmentRepo.GetRoleAssignmentsBySubjects(ctx, user, app, subjects)
	if err != nil {
		return nil, fmt.Errorf("查询角色分配失败: %w", err)
	}

	// 构建权限映射
	rolePermissionMap := NewRolePermissionMap()
	now := time.Now()

	for _, assignment := range roleAssignments {
		// 检查角色是否在有效期内
		if !assignment.IsEffective(now) {
			continue
		}

		// 从内存缓存获取角色权限（按资源类型分组）
		rolePermsByType := l.roleCache.GetRolePermissionsGroupedByResourceType(assignment.RoleID)
		if len(rolePermsByType) == 0 {
			continue
		}

		resourcePath := assignment.ResourcePath

		// 合并角色权限（按资源类型）
		for resourceType, actions := range rolePermsByType {
			for action := range actions {
				rolePermissionMap.Add(resourcePath, resourceType, action)
			}
		}
	}

	return rolePermissionMap, nil
}

// PermissionCalculatorV2 权限计算器 V2（结构化版本）
type PermissionCalculatorV2 struct {
	roleAssignmentLoader *RoleAssignmentLoader
	inheritanceApplier    *PermissionInheritanceApplier
}

// NewPermissionCalculatorV2 创建权限计算器 V2
func NewPermissionCalculatorV2(
	roleAssignmentRepo *repository.RoleAssignmentRepository,
	roleCache *RoleCache,
) *PermissionCalculatorV2 {
	return &PermissionCalculatorV2{
		roleAssignmentLoader: NewRoleAssignmentLoader(roleAssignmentRepo, roleCache),
		inheritanceApplier:    NewPermissionInheritanceApplier(),
	}
}

// CalculateWorkspacePermissions 计算工作空间所有节点的权限（服务树场景）
func (c *PermissionCalculatorV2) CalculateWorkspacePermissions(
	ctx context.Context,
	user string,
	app string,
	username string,
	departmentPath string,
	trees []*model.ServiceTree,
) (map[string]map[string]bool, error) {
	// 1. 构建权限主体列表
	subjects := c.buildSubjects(username, departmentPath)

	// 2. 加载角色分配并构建权限映射
	rolePermissionMap, err := c.roleAssignmentLoader.Load(ctx, user, app, subjects)
	if err != nil {
		return nil, err
	}

	// 3. 计算所有节点的权限
	permissionsMap := make(map[string]map[string]bool)
	c.calculateRecursive(trees, nil, rolePermissionMap, permissionsMap)

	logger.Debugf(ctx, "[PermissionCalculatorV2] 权限计算完成（基于角色）: 节点数=%d, 权限节点数=%d", len(trees), len(permissionsMap))

	return permissionsMap, nil
}

// buildSubjects 构建权限主体列表
func (c *PermissionCalculatorV2) buildSubjects(username string, departmentPath string) []permissionrepo.SubjectInfo {
	subjects := []permissionrepo.SubjectInfo{
		{Type: "user", Value: username}, // 用户本身
	}

	// 添加组织架构路径及其所有父级路径
	if departmentPath != "" {
		deptPaths := getAllParentDeptPathsForRole(departmentPath)
		for _, path := range deptPaths {
			subjects = append(subjects, permissionrepo.SubjectInfo{
				Type:  "department",
				Value: path,
			})
		}
	}

	return subjects
}

// calculateRecursive 递归计算节点权限
func (c *PermissionCalculatorV2) calculateRecursive(
	nodes []*model.ServiceTree,
	inheritedPerms map[string]bool,
	rolePermissionMap *RolePermissionMap,
	permissionsMap map[string]map[string]bool,
) {
	for _, node := range nodes {
		// 获取节点需要的权限点
		requiredActions := permission.GetActionsForNode(node.Type, node.TemplateType)
		if len(requiredActions) == 0 {
			// 如果没有需要的权限点，继续处理子节点
			if len(node.Children) > 0 {
				c.calculateRecursive(node.Children, inheritedPerms, rolePermissionMap, permissionsMap)
			}
			continue
		}

		// 根据节点类型和模板类型获取资源类型
		resourceType := permission.GetResourceType(node.Type, node.TemplateType)

		// 创建节点权限解析器
		resolver := NewNodePermissionResolver(rolePermissionMap, resourceType, requiredActions)

		// 解析节点权限
		nodePerms := resolver.Resolve(node.FullCodePath)

		// 应用继承权限（从父节点传递下来的）
		if inheritedPerms != nil {
			c.inheritanceApplier.Apply(node.Type, node.TemplateType, inheritedPerms, nodePerms)
		}

		// 保存节点权限
		permissionsMap[node.FullCodePath] = nodePerms

		// 计算传递给子节点的继承权限
		childInheritedPerms := make(map[string]bool)
		for k, v := range inheritedPerms {
			childInheritedPerms[k] = v
		}
		// 添加当前节点的权限（用于继承）
		for action, granted := range nodePerms {
			if granted {
				childInheritedPerms[action] = true
			}
		}

		// 递归处理子节点
		if len(node.Children) > 0 {
			c.calculateRecursive(node.Children, childInheritedPerms, rolePermissionMap, permissionsMap)
		}
	}
}

// getUserRolePermissions 获取用户角色权限（统一查询）
func (c *PermissionCalculatorV2) getUserRolePermissions(
	ctx context.Context,
	user string,
	app string,
	username string,
	departmentPath string,
) (map[string]map[string]bool, error) {
	// 1. 构建权限主体列表
	subjects := c.buildSubjects(username, departmentPath)

	// 2. 加载角色分配并构建权限映射
	rolePermissionMap, err := c.roleAssignmentLoader.Load(ctx, user, app, subjects)
	if err != nil {
		return nil, err
	}

	// 3. 转换为返回格式：resourcePath -> action -> true
	permissions := make(map[string]map[string]bool)
	for resourcePath, resourceTypeMap := range rolePermissionMap.data {
		if permissions[resourcePath] == nil {
			permissions[resourcePath] = make(map[string]bool)
		}
		// 合并所有资源类型的权限
		for _, actionMap := range resourceTypeMap {
			for action := range actionMap {
				permissions[resourcePath][action] = true
			}
		}
	}

	return permissions, nil
}

// getAppPath 获取应用根路径（/user/app）
func getAppPath(fullCodePath string) string {
	parts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(parts) >= 2 {
		return "/" + strings.Join(parts[:2], "/")
	}
	return ""
}
