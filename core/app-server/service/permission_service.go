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
	permissionService  enterprise.PermissionService
	serviceTreeService *ServiceTreeService // ⭐ 添加 ServiceTreeService 依赖，用于获取工作空间权限
}

// NewPermissionService 创建权限管理服务
func NewPermissionService(permissionService enterprise.PermissionService, serviceTreeService *ServiceTreeService) *PermissionService {
	return &PermissionService{
		permissionService:  permissionService,
		serviceTreeService: serviceTreeService,
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
				"function:read", "table:write", "table:update", "table:delete",
				"form:write", "function:manage",
				"directory:read", "directory:write", "directory:update", "directory:delete", "directory:manage",
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

// GetWorkspacePermissions 获取工作空间的所有权限
// 遍历整个服务树，批量查询所有节点的权限
func (s *PermissionService) GetWorkspacePermissions(ctx context.Context, req *dto.GetWorkspacePermissionsReq) (*dto.GetWorkspacePermissionsResp, error) {
	resp := &dto.GetWorkspacePermissionsResp{
		User:        req.User,
		App:         req.App,
		Permissions: make(map[string]map[string]bool),
	}

	// 1. 获取服务树结构
	serviceTreeResp, err := s.serviceTreeService.GetServiceTree(ctx, req.User, req.App, "")
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 获取服务树失败: user=%s, app=%s, error=%v",
			req.User, req.App, err)
		return nil, fmt.Errorf("获取服务树失败: %w", err)
	}

	// 2. 递归收集所有节点的路径和类型信息
	type nodeInfo struct {
		fullCodePath string
		nodeType     string
		templateType string
	}
	var allNodes []nodeInfo

	var collectNodes func(nodes []*dto.GetServiceTreeResp)
	collectNodes = func(nodes []*dto.GetServiceTreeResp) {
		for _, node := range nodes {
			allNodes = append(allNodes, nodeInfo{
				fullCodePath: node.FullCodePath,
				nodeType:     node.Type,
				templateType: node.TemplateType,
			})
			if len(node.Children) > 0 {
				collectNodes(node.Children)
			}
		}
	}
	collectNodes(serviceTreeResp)

	// 3. 为每个节点确定需要查询的权限点
	resourcePaths := make([]string, 0, len(allNodes))
	resourceActions := make(map[string][]string) // resourcePath -> actions

	for _, node := range allNodes {
		actions := s.getPermissionActionsForNode(node.nodeType, node.templateType)
		resourcePaths = append(resourcePaths, node.fullCodePath)
		resourceActions[node.fullCodePath] = actions
	}

	// 4. 收集所有需要查询的权限点（去重）
	allActionsMap := make(map[string]bool)
	for _, actions := range resourceActions {
		for _, action := range actions {
			allActionsMap[action] = true
		}
	}
	allActions := make([]string, 0, len(allActionsMap))
	for action := range allActionsMap {
		allActions = append(allActions, action)
	}

	// 5. 批量查询所有节点的权限
	permissions, err := s.permissionService.BatchCheckPermissions(ctx, req.User, resourcePaths, allActions)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 批量查询权限失败: user=%s, app=%s, error=%v",
			req.User, req.App, err)
		return nil, fmt.Errorf("批量查询权限失败: %w", err)
	}

	// 6. 整理结果：只返回每个节点相关的权限点，并且只返回有权限的节点
	for _, node := range allNodes {
		nodePerms := make(map[string]bool)
		actions := resourceActions[node.fullCodePath]
		if nodePermsFromBatch, ok := permissions[node.fullCodePath]; ok {
			// 只保留该节点相关的权限点
			for _, action := range actions {
				if hasPerm, exists := nodePermsFromBatch[action]; exists {
					nodePerms[action] = hasPerm
				} else {
					nodePerms[action] = false
				}
			}
		} else {
			// 如果批量查询结果中没有该节点，初始化所有权限为 false
			for _, action := range actions {
				nodePerms[action] = false
			}
		}
		
		// ⭐ 只返回有权限的节点（至少有一个权限为 true）
		hasAnyPermission := false
		for _, hasPerm := range nodePerms {
			if hasPerm {
				hasAnyPermission = true
				break
			}
		}
		
		if hasAnyPermission {
			resp.Permissions[node.fullCodePath] = nodePerms
		}
		// 如果没有权限，不添加到结果中，减少返回数据量
	}

	return resp, nil
}

// getPermissionActionsForNode 根据节点类型和模板类型，获取需要检查的权限点
// 与 ServiceTreeService 中的方法保持一致
func (s *PermissionService) getPermissionActionsForNode(nodeType string, templateType string) []string {
	actions := make([]string, 0)

	if nodeType == "package" {
		// 服务目录（package）：检查目录权限
		actions = append(actions,
			"directory:read",
			"directory:write",
			"directory:update",
			"directory:delete",
			"directory:manage",
		)
	} else if nodeType == "function" {
		// 函数（function）：检查函数权限和操作级别权限
		actions = append(actions,
			"function:read",
			"function:manage",
		)

		// 根据模板类型，添加操作级别的权限点
		switch templateType {
		case "table":
			actions = append(actions,
				"table:write",
				"table:update",
				"table:delete",
			)
		case "form":
			actions = append(actions,
				"form:write",
			)
		case "chart":
			// chart 使用 function:read 权限，拥有 read 权限即视为拥有 query 权限
		}
	}

	return actions
}

