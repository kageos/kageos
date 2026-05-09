package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

func (a *AppService) grantCreateAppAdmins(ctx context.Context, user, app, admins string) {
	resourcePath := fmt.Sprintf("/%s/%s", user, app)

	if err := a.assignAppAdminRoleToUser(ctx, user, app, user, resourcePath); err != nil {
		logger.Warnf(ctx, "[AppService] 自动添加创建者应用管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
			user, app, user, resourcePath, err)
	}

	if admins == "" {
		return
	}

	for _, admin := range strings.Split(admins, ",") {
		admin = strings.TrimSpace(admin)
		if admin == "" || admin == user {
			continue
		}
		if err := a.assignAppAdminRoleToUser(ctx, user, app, admin, resourcePath); err != nil {
			logger.Warnf(ctx, "[AppService] 自动添加应用管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				user, app, admin, resourcePath, err)
		}
	}
}

// assignAppAdminRoleToUser 给用户分配应用管理员角色。
// 使用角色系统，分配 "admin" 角色（拥有 app:admin 权限）。
func (a *AppService) assignAppAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	assignReq := &dto.AssignRoleToUserReq{
		Username:     username,
		RoleCode:     permission.RoleCodeAdmin,
		ResourceType: permission.ResourceTypeDirectory,
		ResourcePath: resourcePath,
		StartTime:    nil,
		EndTime:      nil,
	}

	if _, err := permissionService.AssignRoleToUser(ctx, assignReq); err != nil {
		return fmt.Errorf("分配应用管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 分配应用管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

// removeAppAdminRoleFromUser 移除用户的应用管理员角色。
func (a *AppService) removeAppAdminRoleFromUser(ctx context.Context, user, app, username, resourcePath string) error {
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	removeReq := &dto.RemoveRoleFromUserReq{
		Username:     username,
		RoleCode:     permission.RoleCodeAdmin,
		ResourceType: permission.ResourceTypeDirectory,
		ResourcePath: resourcePath,
	}

	if err := permissionService.RemoveRoleFromUser(ctx, removeReq); err != nil {
		return fmt.Errorf("移除应用管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 移除应用管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}
