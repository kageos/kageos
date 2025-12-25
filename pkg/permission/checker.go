package permission

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// CheckPermissionWithInheritance 检查权限（支持权限继承）
// 会检查：
// 1. 当前资源的直接权限
// 2. 所有父目录的 directory:manage 权限（权限继承）
//
// 参数：
//   - ctx: 上下文
//   - permissionService: 权限服务
//   - username: 用户名
//   - fullCodePath: 资源路径（full-code-path）
//   - action: 操作类型（如 function:read、table:create 等）
//
// 返回：
//   - hasPermission: 是否有权限
//   - err: 错误信息
func CheckPermissionWithInheritance(
	ctx context.Context,
	permissionService enterprise.PermissionService,
	username string,
	fullCodePath string,
	action string,
) (hasPermission bool, err error) {
	// 构建所有需要检查的权限点（批量查询）
	// 1. 当前资源的权限
	// 2. 所有父目录的 directory:manage 权限（用于权限继承）
	resourcePaths := []string{fullCodePath}
	actions := []string{action}

	// 解析路径，获取所有父目录路径
	parentPaths := GetParentPaths(fullCodePath)
	for _, parentPath := range parentPaths {
		resourcePaths = append(resourcePaths, parentPath)
		actions = append(actions, "directory:manage")
	}

	// 批量查询所有权限
	permissions, err := permissionService.BatchCheckPermissions(ctx, username, resourcePaths, actions)
	if err != nil {
		logger.Warnf(ctx, "[PermissionChecker] 批量权限查询失败: resource=%s, username=%s, error=%v",
			fullCodePath, username, err)
		return false, err
	}

	// 按优先级判断权限（先检查当前资源，再检查父目录）
	// 1. 优先检查当前资源的直接权限
	if nodePerms, ok := permissions[fullCodePath]; ok {
		if hasPerm, ok := nodePerms[action]; ok && hasPerm {
			logger.Debugf(ctx, "[PermissionChecker] 直接权限通过: resource=%s, action=%s", fullCodePath, action)
			return true, nil
		}
	}

	// 2. 如果直接权限失败，检查父目录的 directory:manage 权限（权限继承）
	for _, parentPath := range parentPaths {
		if nodePerms, ok := permissions[parentPath]; ok {
			if hasManage, ok := nodePerms["directory:manage"]; ok && hasManage {
				// 父目录有管理权限，自动拥有所有子资源的权限
				logger.Infof(ctx, "[PermissionChecker] 权限继承成功: resource=%s, action=%s, parent=%s, has directory:manage",
					fullCodePath, action, parentPath)
				return true, nil
			}
		}
	}

	// 所有检查都失败，返回无权限
	return false, nil
}

// BuildPermissionCheckRequests 构建权限检查请求（用于批量查询）
// 返回所有需要检查的 resourcePath 和 action 对
func BuildPermissionCheckRequests(fullCodePath string, action string) (resourcePaths []string, actions []string) {
	resourcePaths = []string{fullCodePath}
	actions = []string{action}

	// 解析路径，获取所有父目录路径
	parentPaths := GetParentPaths(fullCodePath)
	for _, parentPath := range parentPaths {
		resourcePaths = append(resourcePaths, parentPath)
		actions = append(actions, "directory:manage")
	}

	return resourcePaths, actions
}

