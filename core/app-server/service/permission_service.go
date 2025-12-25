package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// PermissionService 权限管理服务
type PermissionService struct {
	permissionService enterprise.PermissionService
}

// NewPermissionService 创建权限管理服务
func NewPermissionService(permissionService enterprise.PermissionService) *PermissionService {
	return &PermissionService{
		permissionService: permissionService,
	}
}

// AddPermission 添加权限
func (s *PermissionService) AddPermission(ctx context.Context, req *dto.AddPermissionReq) error {
	// 直接调用接口方法（不需要类型断言）
	err := s.permissionService.AddPolicy(ctx, req.Username, req.ResourcePath, req.Action)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 添加权限失败: user=%s, resource=%s, action=%s, error=%v",
			req.Username, req.ResourcePath, req.Action, err)
		return fmt.Errorf("添加权限失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 添加权限成功: user=%s, resource=%s, action=%s",
		req.Username, req.ResourcePath, req.Action)
	return nil
}

// RemovePermission 删除权限
func (s *PermissionService) RemovePermission(ctx context.Context, req *dto.RemovePermissionReq) error {
	// 直接调用接口方法（不需要类型断言）
	err := s.permissionService.RemovePolicy(ctx, req.Username, req.ResourcePath, req.Action)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 删除权限失败: user=%s, resource=%s, action=%s, error=%v",
			req.Username, req.ResourcePath, req.Action, err)
		return fmt.Errorf("删除权限失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 删除权限成功: user=%s, resource=%s, action=%s",
		req.Username, req.ResourcePath, req.Action)
	return nil
}

// GetUserPermissions 获取用户权限
func (s *PermissionService) GetUserPermissions(ctx context.Context, req *dto.GetUserPermissionsReq) (*dto.GetUserPermissionsResp, error) {
	resp := &dto.GetUserPermissionsResp{
		Username:     req.Username,
		ResourcePath: req.ResourcePath,
		Permissions:  make(map[string]bool),
	}

	// 如果指定了资源路径，只查询该资源的权限
	if req.ResourcePath != "" {
		// 如果指定了操作列表，只查询这些操作的权限
		if len(req.Actions) > 0 {
			resourcePaths := []string{req.ResourcePath}
			permissions, err := s.permissionService.BatchCheckPermissions(ctx, req.Username, resourcePaths, req.Actions)
			if err != nil {
				logger.Errorf(ctx, "[PermissionService] 查询用户权限失败: user=%s, resource=%s, error=%v",
					req.Username, req.ResourcePath, err)
				return nil, fmt.Errorf("查询用户权限失败: %w", err)
			}

			if nodePerms, ok := permissions[req.ResourcePath]; ok {
				resp.Permissions = nodePerms
			}
		} else {
			// 如果没有指定操作列表，查询所有常见操作的权限
			actions := []string{
				"table:search", "table:create", "table:update", "table:delete",
				"form:submit", "chart:query", "callback:on_select_fuzzy",
				"function:read", "function:execute",
				"directory:read", "directory:create", "directory:update", "directory:delete", "directory:manage",
				"app:read", "app:create", "app:update", "app:delete", "app:manage",
			}
			resourcePaths := []string{req.ResourcePath}
			permissions, err := s.permissionService.BatchCheckPermissions(ctx, req.Username, resourcePaths, actions)
			if err != nil {
				logger.Errorf(ctx, "[PermissionService] 查询用户权限失败: user=%s, resource=%s, error=%v",
					req.Username, req.ResourcePath, err)
				return nil, fmt.Errorf("查询用户权限失败: %w", err)
			}

			if nodePerms, ok := permissions[req.ResourcePath]; ok {
				resp.Permissions = nodePerms
			}
		}
	} else {
		// 如果没有指定资源路径，返回空结果（或者可以查询所有资源，但这可能性能较差）
		logger.Warnf(ctx, "[PermissionService] 未指定资源路径，返回空权限结果")
	}

	return resp, nil
}

// AssignRoleToUser 分配角色给用户
func (s *PermissionService) AssignRoleToUser(ctx context.Context, req *dto.AssignRoleToUserReq) error {
	// 直接调用接口方法（不需要类型断言）
	err := s.permissionService.AddGroupingPolicy(ctx, req.Username, req.RoleName)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 分配角色失败: user=%s, role=%s, error=%v",
			req.Username, req.RoleName, err)
		return fmt.Errorf("分配角色失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 分配角色成功: user=%s, role=%s",
		req.Username, req.RoleName)
	return nil
}

// RemoveRoleFromUser 从用户移除角色
func (s *PermissionService) RemoveRoleFromUser(ctx context.Context, req *dto.RemoveRoleFromUserReq) error {
	// 直接调用接口方法（不需要类型断言）
	err := s.permissionService.RemoveGroupingPolicy(ctx, req.Username, req.RoleName)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 移除角色失败: user=%s, role=%s, error=%v",
			req.Username, req.RoleName, err)
		return fmt.Errorf("移除角色失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 移除角色成功: user=%s, role=%s",
		req.Username, req.RoleName)
	return nil
}

// GetUserRoles 获取用户角色
func (s *PermissionService) GetUserRoles(ctx context.Context, username string) (*dto.GetUserRolesResp, error) {
	// 直接调用接口方法（不需要类型断言）
	roles, err := s.permissionService.GetRolesForUser(ctx, username)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 查询用户角色失败: user=%s, error=%v",
			username, err)
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}

	return &dto.GetUserRolesResp{
		Username: username,
		Roles:    roles,
	}, nil
}

