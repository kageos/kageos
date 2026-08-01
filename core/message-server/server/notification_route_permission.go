package server

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apicall"
)

type notificationRoutePermissionResolver interface {
	MyPermissions(ctx context.Context, resourcePath string) (*dto.MyPermissionsResp, error)
}

type workspaceNotificationRoutePermissionResolver struct{}

func (workspaceNotificationRoutePermissionResolver) MyPermissions(ctx context.Context, resourcePath string) (*dto.MyPermissionsResp, error) {
	return apicall.MyPermissions(ctx, resourcePath)
}

func (s *Server) requireNotificationRouteAdmin(ctx context.Context, scopePath string) error {
	scopePath = access.NormalizeResourcePath(scopePath)
	if scopePath == "" {
		return fmt.Errorf("scope_path 不能为空")
	}
	if s == nil || s.notificationRouteAuth == nil {
		return fmt.Errorf("目录通知权限服务未初始化")
	}

	permissions, err := s.notificationRouteAuth.MyPermissions(ctx, scopePath)
	if err != nil {
		return fmt.Errorf("校验目录通知管理权限失败: %w", err)
	}
	if permissions == nil || !access.HasPermission(permissions.Permissions, access.ActionAdmin) {
		return fmt.Errorf("无权限管理目录通知配置，需要 admin 权限: %s", scopePath)
	}
	return nil
}
