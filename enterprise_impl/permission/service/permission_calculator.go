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

// hasAnyPermission 检查权限映射中是否有任何权限为 true
func hasAnyPermission(perms map[string]bool) bool {
	for _, hasPerm := range perms {
		if hasPerm {
			return true
		}
	}
	return false
}

// PermissionCalculator 权限计算器（企业版，仅使用角色系统）
type PermissionCalculator struct {
	roleAssignmentRepo *repository.RoleAssignmentRepository
	roleCache          *RoleCache
}

// NewPermissionCalculator 创建权限计算器
func NewPermissionCalculator(
	roleAssignmentRepo *repository.RoleAssignmentRepository,
	roleCache *RoleCache,
) *PermissionCalculator {
	return &PermissionCalculator{
		roleAssignmentRepo: roleAssignmentRepo,
		roleCache:          roleCache,
	}
}

// CalculateWorkspacePermissions 计算工作空间所有节点的权限（服务树场景）
// ⭐ 优化：一次性查询用户在该工作空间的所有角色分配，根据角色配置直接给节点赋值权限
func (c *PermissionCalculator) CalculateWorkspacePermissions(
	ctx context.Context,
	user string,
	app string,
	username string,
	departmentPath string,
	trees []*model.ServiceTree, // 服务树节点列表
) (map[string]map[string]bool, error) {
	// 1. ⭐ 一次性查询用户在该工作空间的所有角色分配
	// 包括：
	//   - 用户本身的角色分配
	//   - 用户所属组织架构的角色分配
	//   - 用户所属组织架构的所有父级组织架构的角色分配
	subjects := []permissionrepo.SubjectInfo{
		{Type: "user", Value: username}, // 用户本身
	}

	// 添加组织架构路径及其所有父级路径
	// 例如：/org/master/bizit → ["/org/master/bizit", "/org/master", "/org"]
	if departmentPath != "" {
		deptPaths := getAllParentDeptPathsForRole(departmentPath)
		for _, path := range deptPaths {
			subjects = append(subjects, permissionrepo.SubjectInfo{
				Type:  "department",
				Value: path,
			})
		}
	}

	// ⭐ 一次查询获取所有角色分配（用户 + 组织架构 + 父级组织架构）
	roleAssignments, err := c.roleAssignmentRepo.GetRoleAssignmentsBySubjects(ctx, user, app, subjects)
	if err != nil {
		return nil, fmt.Errorf("查询角色分配失败: %w", err)
	}

	// 2. ⭐ 构建角色权限映射：resourcePath -> resourceType -> actions
	// 例如：{"/user/app/dir": {"directory": {"read": true, "write": true}}}
	// ⭐ 同时支持通配符路径匹配（如 /user/app/dir/* 匹配所有子路径）
	rolePermissionsByPath := make(map[string]map[string]map[string]bool) // resourcePath -> resourceType -> action -> true

	now := time.Now()
	for _, assignment := range roleAssignments {
		// 检查角色是否在有效期内
		if !assignment.IsEffective(now) {
			continue
		}

		// ⭐ 从内存缓存获取角色权限（按资源类型分组）
		rolePermsByType := c.roleCache.GetRolePermissionsGroupedByResourceType(assignment.RoleID)
		if len(rolePermsByType) == 0 {
			continue
		}

		resourcePath := assignment.ResourcePath
		if rolePermissionsByPath[resourcePath] == nil {
			rolePermissionsByPath[resourcePath] = make(map[string]map[string]bool)
		}

		// 合并角色权限（按资源类型）
		for resourceType, actions := range rolePermsByType {
			if rolePermissionsByPath[resourcePath][resourceType] == nil {
				rolePermissionsByPath[resourcePath][resourceType] = make(map[string]bool)
			}
			for action := range actions {
				rolePermissionsByPath[resourcePath][resourceType][action] = true
			}
		}
	}

	// 3. ⭐ 遍历所有节点，根据角色配置直接赋值权限
	permissionsMap := make(map[string]map[string]bool)

	var calculateRecursive func(nodes []*model.ServiceTree, inheritedPerms map[string]bool)
	calculateRecursive = func(nodes []*model.ServiceTree, inheritedPerms map[string]bool) {
		for _, node := range nodes {
			// 获取节点需要的权限点
			requiredActions := permission.GetActionsForNode(node.Type, node.TemplateType)
			if len(requiredActions) == 0 {
				// 如果没有需要的权限点，继续处理子节点
				if len(node.Children) > 0 {
					calculateRecursive(node.Children, inheritedPerms)
				}
				continue
			}

			// 初始化节点权限
			nodePerms := make(map[string]bool)
			for _, action := range requiredActions {
				nodePerms[action] = false
			}

			// ⭐ 根据节点类型和模板类型获取资源类型
			resourceType := permission.GetResourceType(node.Type, node.TemplateType)

			// ⭐ 查找匹配的角色权限（支持精确匹配和目录继承）
			// 1. 精确匹配：node.FullCodePath
			// 2. 目录继承：parentPath（目录权限自动继承给子节点）

			// ⭐ 权限点格式：resource_type:action_type（如 form:read, table:write）
			// 检查精确路径
			if rolePermsByType, ok := rolePermissionsByPath[node.FullCodePath]; ok {
				// 检查该资源类型的权限
				if rolePerms, ok := rolePermsByType[resourceType]; ok {
					for actionCode := range rolePerms {
						// 直接匹配权限点编码（resource_type:action_type）
						for _, requiredAction := range requiredActions {
							if actionCode == requiredAction {
								nodePerms[requiredAction] = true
								break
							}
						}
					}
				}
			}

			// 检查父目录路径（目录权限继承）
			parentPaths := permission.GetParentPaths(node.FullCodePath)
			for _, parentPath := range parentPaths {
				// ⭐ 检查精确路径匹配
				if rolePermsByType, ok := rolePermissionsByPath[parentPath]; ok {
					// ⭐ 目录权限继承：directory:read -> table:read（需要转换）
					if directoryPerms, ok := rolePermsByType[permission.ResourceTypeDirectory]; ok {
						for directoryActionCode := range directoryPerms {
							// 解析目录权限点编码
							_, actionType, ok := permission.ParseActionCode(directoryActionCode)
							if !ok {
								continue
							}

							// 如果是 admin 权限，直接给所有权限
							if actionType == "admin" {
								for _, requiredAction := range requiredActions {
									nodePerms[requiredAction] = true
								}
							} else {
								// 构建该资源类型的权限点编码（directory:read -> table:read）
								functionActionCode := permission.BuildActionCode(resourceType, actionType)
								for _, requiredAction := range requiredActions {
									if functionActionCode == requiredAction {
										nodePerms[requiredAction] = true
										break
									}
								}
							}
						}
					}
					// 检查该资源类型的权限（直接匹配）
					if rolePerms, ok := rolePermsByType[resourceType]; ok {
						for actionCode := range rolePerms {
							for _, requiredAction := range requiredActions {
								if actionCode == requiredAction {
									nodePerms[requiredAction] = true
									break
								}
							}
						}
					}
				}
			}

			// ⭐ 额外检查：如果父目录路径检查失败，尝试前缀匹配（支持通配符路径）
			// 例如：如果角色分配在 /user/app/dir，函数在 /user/app/dir/subdir/function
			// 需要检查 /user/app/dir 是否是函数路径的前缀
			if len(parentPaths) == 0 || !hasAnyPermission(nodePerms) {
				for assignedPath, rolePermsByType := range rolePermissionsByPath {
					// ⭐ 检查 assignedPath 是否是 node.FullCodePath 的前缀（目录权限继承）
					if strings.HasPrefix(node.FullCodePath, assignedPath+"/") || node.FullCodePath == assignedPath {
						// ⭐ 目录权限继承：directory:read -> table:read（需要转换）
						if directoryPerms, ok := rolePermsByType[permission.ResourceTypeDirectory]; ok {
							for directoryActionCode := range directoryPerms {
								// 解析目录权限点编码
								_, actionType, ok := permission.ParseActionCode(directoryActionCode)
								if !ok {
									continue
								}

								// 如果是 admin 权限，直接给所有权限
								if actionType == "admin" {
									for _, requiredAction := range requiredActions {
										nodePerms[requiredAction] = true
									}
								} else {
									// 构建该资源类型的权限点编码（directory:read -> table:read）
									functionActionCode := permission.BuildActionCode(resourceType, actionType)
									for _, requiredAction := range requiredActions {
										if functionActionCode == requiredAction {
											nodePerms[requiredAction] = true
											break
										}
									}
								}
							}
						}
						// 检查该资源类型的权限（直接匹配）
						if rolePerms, ok := rolePermsByType[resourceType]; ok {
							for actionCode := range rolePerms {
								for _, requiredAction := range requiredActions {
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

			// 检查应用级别权限
			appPath := c.getAppPath(node.FullCodePath)
			if appPath != "" {
				if rolePermsByType, ok := rolePermissionsByPath[appPath]; ok {
					if appPerms, ok := rolePermsByType[permission.ResourceTypeApp]; ok {
						appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
						if appPerms[appAdminCode] {
							// app:admin -> 所有权限
							for _, requiredAction := range requiredActions {
								nodePerms[requiredAction] = true
							}
						}
					}
				}
			}

			// 应用继承权限（从父节点传递下来的）
			if inheritedPerms != nil {
				c.applyPermissionInheritance(node.Type, node.TemplateType, inheritedPerms, nodePerms)
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
				calculateRecursive(node.Children, childInheritedPerms)
			}
		}
	}

	// 从根节点开始计算
	calculateRecursive(trees, nil)

	logger.Debugf(ctx, "[PermissionCalculator] 权限计算完成（基于角色）: 节点数=%d, 权限节点数=%d", len(trees), len(permissionsMap))

	return permissionsMap, nil
}

// applyPermissionInheritance 应用权限继承规则（简化版）
// ⭐ 权限点已简化：read, write, update, delete, admin
// 不再需要区分 directory:read 和 function:read，直接继承即可
func (c *PermissionCalculator) applyPermissionInheritance(
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
	// 因为权限点已简化，read 就是 read，不需要区分 directory:read 和 function:read
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

// getAppPath 获取应用根路径（/user/app）
func (c *PermissionCalculator) getAppPath(fullCodePath string) string {
	parts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(parts) >= 2 {
		return "/" + strings.Join(parts[:2], "/")
	}
	return ""
}

// ⭐ 已移除 inferResourceType：不能从路径推断资源类型，因为层级可能很深
// 资源类型应该从 ServiceTree 或 Function 表查询，或者在保存权限时传入

// PermissionContext 权限上下文
type PermissionContext struct {
	Username       string
	DepartmentPath string
	User           string // 租户用户名（从 resource_path 解析）
	App            string // 应用代码（从 resource_path 解析）
}

// GetSubjectList 获取所有权限主体列表
func (ctx *PermissionContext) GetSubjectList() []permissionrepo.SubjectInfo {
	subjects := []permissionrepo.SubjectInfo{
		{Type: "user", Value: ctx.Username},
	}

	if ctx.DepartmentPath != "" {
		// 添加组织架构路径及其所有父级路径
		deptPaths := getAllParentDeptPaths(ctx.DepartmentPath)
		for _, path := range deptPaths {
			subjects = append(subjects, permissionrepo.SubjectInfo{
				Type:  "department",
				Value: path,
			})
		}
	}

	return subjects
}

// getAllParentDeptPaths 获取所有父级组织架构路径
func getAllParentDeptPaths(deptPath string) []string {
	parts := strings.Split(strings.Trim(deptPath, "/"), "/")
	if len(parts) == 0 {
		return []string{}
	}

	paths := make([]string, 0, len(parts))
	for i := len(parts); i >= 1; i-- {
		path := "/" + strings.Join(parts[:i], "/")
		paths = append(paths, path)
	}
	return paths
}

// ResourceNode 资源节点
type ResourceNode struct {
	Path       string
	Type       string
	ParentPath string
	Depth      int
}

// GetParentPaths 获取所有父节点路径
func (n *ResourceNode) GetParentPaths() []string {
	if n.Depth <= 1 {
		return []string{}
	}

	parts := strings.Split(strings.Trim(n.Path, "/"), "/")
	if len(parts) < 2 {
		return []string{}
	}

	parentPaths := make([]string, 0, len(parts)-1)
	for i := len(parts) - 1; i >= 2; i-- {
		parentPath := "/" + strings.Join(parts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}
	return parentPaths
}

// GetAppPath 获取应用根路径
func (n *ResourceNode) GetAppPath() string {
	parts := strings.Split(strings.Trim(n.Path, "/"), "/")
	if len(parts) >= 2 {
		return "/" + strings.Join(parts[:2], "/")
	}
	return ""
}
