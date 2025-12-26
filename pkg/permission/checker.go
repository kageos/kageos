package permission

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// CheckPermissionWithInheritance 检查权限（支持权限继承）
// ⭐ 优化：权限继承现在由 Casbin Matcher 自动处理（通过 keyMatch2 和权限映射）
// 所以这里只需要检查当前资源的权限，Casbin 会自动处理父目录的权限继承
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
//
// 说明：
//   - Casbin Matcher 已经支持权限映射：
//     * directory:manage 权限自动覆盖所有子资源的权限
//     * app:manage 权限自动覆盖应用下所有资源的权限
//   - 所以这里只需要调用 Casbin 的 CheckPermission，它会自动处理权限继承
func CheckPermissionWithInheritance(
	ctx context.Context,
	permissionService enterprise.PermissionService,
	username string,
	fullCodePath string,
	action string,
) (hasPermission bool, err error) {
	// ⭐ 优化：直接调用 Casbin 的 CheckPermission
	// Casbin Matcher 已经支持权限映射和路径匹配，会自动处理权限继承
	hasPermission, err = permissionService.CheckPermission(ctx, username, fullCodePath, action)
	if err != nil {
		logger.Warnf(ctx, "[PermissionChecker] 权限检查失败: resource=%s, username=%s, action=%s, error=%v",
			fullCodePath, username, action, err)
		return false, err
	}

	if hasPermission {
		logger.Debugf(ctx, "[PermissionChecker] 权限检查通过: resource=%s, action=%s", fullCodePath, action)
	} else {
		logger.Debugf(ctx, "[PermissionChecker] 权限检查失败: resource=%s, action=%s", fullCodePath, action)
	}

	return hasPermission, nil
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

