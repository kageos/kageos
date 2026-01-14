package service

import (
	"context"
	"strings"
	"time"

	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
)

// getUserRolePermissions 获取用户角色权限（统一查询）
func (c *PermissionCalculator) getUserRolePermissions(
	ctx context.Context,
	user string,
	app string,
	username string,
	departmentPath string,
) (map[string]map[string]bool, error) {
	// 1. 构建权限主体列表（用户 + 组织架构路径）
	subjects := []permissionrepo.SubjectInfo{
		{Type: "user", Value: username},
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

	// 2. ⭐ 一次查询获取所有角色分配（用户 + 组织架构）
	roleAssignments, err := c.roleAssignmentRepo.GetRoleAssignmentsBySubjects(ctx, user, app, subjects)
	if err != nil {
		return nil, err
	}

	// 3. 从内存缓存获取角色权限
	permissions := make(map[string]map[string]bool)

	for _, assignment := range roleAssignments {
		// 检查角色是否在有效期内
		if !assignment.IsEffective(time.Now()) {
			continue
		}

		// ⭐ 从内存缓存获取角色权限（快速）
		// 权限点格式：resource_type:action_type（如 directory:read, table:write）
		// 返回所有资源类型的权限点编码
		rolePerms := c.roleCache.GetRolePermissions(assignment.RoleID)
		if len(rolePerms) == 0 {
			continue
		}

		// 检查资源路径是否匹配（支持通配符）
		resourcePath := assignment.ResourcePath
		if permissions[resourcePath] == nil {
			permissions[resourcePath] = make(map[string]bool)
		}

		// ⭐ 添加角色权限（权限点编码格式：resource_type:action_type）
		for actionCode := range rolePerms {
			permissions[resourcePath][actionCode] = true
		}
	}

	return permissions, nil
}

// getAllParentDeptPathsForRole 获取组织架构路径及其所有父级路径（角色系统专用）
// 例如：/org/master/bizit → ["/org/master/bizit", "/org/master", "/org"]
func getAllParentDeptPathsForRole(departmentPath string) []string {
	if departmentPath == "" {
		return []string{}
	}

	// 移除开头的 /org
	path := departmentPath
	if strings.HasPrefix(path, "/org") {
		path = path[4:] // 移除 "/org"
	}

	// 分割路径
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return []string{}
	}

	// 构建所有父级路径
	parentPaths := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		parentPath := "/org" + "/" + strings.Join(parts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}

	return parentPaths
}
