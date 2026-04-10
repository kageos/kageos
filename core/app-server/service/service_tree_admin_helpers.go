package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func parseAdminUserSet(admins string) map[string]bool {
	result := make(map[string]bool)
	for _, admin := range strings.Split(admins, ",") {
		admin = strings.TrimSpace(admin)
		if admin != "" {
			result[admin] = true
		}
	}
	return result
}

func parseUserAppFromResourcePath(resourcePath string) (string, string, error) {
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("无法从资源路径解析 user 和 app: %s", resourcePath)
	}
	return parts[0], parts[1], nil
}

func assignDirectoryAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		return nil
	}

	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	assignReq := &dto.AssignRoleToUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin",
		ResourceType: "directory",
		ResourcePath: resourcePath,
		StartTime:    nil,
		EndTime:      nil,
	}

	if _, err := permissionService.AssignRoleToUser(ctx, assignReq); err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTree] 分配管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

func removeDirectoryAdminRoleFromUser(ctx context.Context, resourcePath, username string) error {
	user, app, err := parseUserAppFromResourcePath(resourcePath)
	if err != nil {
		return err
	}
	return removeDirectoryAdminRoleFromUserWithUserApp(ctx, user, app, username, resourcePath)
}

func removeDirectoryAdminRoleFromUserWithUserApp(ctx context.Context, user, app, username, resourcePath string) error {
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		return nil
	}

	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	removeReq := &dto.RemoveRoleFromUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin",
		ResourceType: "directory",
		ResourcePath: resourcePath,
	}

	if err := permissionService.RemoveRoleFromUser(ctx, removeReq); err != nil {
		return fmt.Errorf("移除管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTree] 移除管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}
