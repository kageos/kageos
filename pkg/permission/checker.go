package permission

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// CheckPermissionWithInheritance 检查权限（支持权限继承）
// ⭐ 使用新的权限系统，支持权限继承
// 新权限系统会自动检查父目录权限并应用继承逻辑
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
//   - 新权限系统支持权限继承：
//     * directory:manage 权限自动覆盖所有子资源的权限
//     * app:manage 权限自动覆盖应用下所有资源的权限
//   - 调用 CheckPermission 会自动处理权限继承
func CheckPermissionWithInheritance(
	ctx context.Context,
	permissionService enterprise.PermissionService,
	username string,
	fullCodePath string,
	action string,
) (hasPermission bool, err error) {
	// ⭐ 使用新的权限系统，自动处理权限继承
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

