package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// appIDResolver 实现 enterprise.AppIDResolver，从 resource_path 解析 app_id（依赖 app-server 的 appRepo）
type appIDResolver struct {
	appRepo *repository.AppRepository
}

func (r *appIDResolver) GetAppIDFromResourcePath(ctx context.Context, resourcePath string) (int64, error) {
	_, user, app := permission.ParseFullCodePath(resourcePath)
	if user == "" || app == "" {
		return 0, fmt.Errorf("资源路径格式错误，无法解析 user 和 app: %s", resourcePath)
	}
	appModel, err := r.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return 0, fmt.Errorf("查询应用失败: user=%s, app=%s: %w", user, app, err)
	}
	return appModel.ID, nil
}

// NewAppIDResolver 创建 AppIDResolver（供 server 在初始化企业版时注入到 enterprise.InitOptions）
func NewAppIDResolver(appRepo *repository.AppRepository) enterprise.AppIDResolver {
	return &appIDResolver{appRepo: appRepo}
}

// PermissionService provides the narrow permission adapter still needed by
// service-tree permission calculation. Permission request, approval, role, and
// grant-management HTTP APIs have been retired from app-server.
type PermissionService struct {
	getPermissionService func() enterprise.PermissionService
}

// NewPermissionService 创建权限管理服务
func NewPermissionService() *PermissionService {
	return &PermissionService{
		getPermissionService: enterprise.GetPermissionService,
	}
}

func (s *PermissionService) permissionBackend() enterprise.PermissionService {
	if s != nil && s.getPermissionService != nil {
		return s.getPermissionService()
	}
	return enterprise.GetPermissionService()
}

// GetWorkspacePermissions 获取工作空间的所有权限
// ⭐ 优化：支持查询用户权限和组织架构权限（v0 可以是用户名或组织架构路径）
// ⭐ 一次性查询用户及其组织架构的所有权限，性能更好
// ⭐ 支持传递用户和组织架构参数，使方法可复用（既可以获取当前用户权限，也可以获取其他用户权限）
func (s *PermissionService) GetWorkspacePermissions(ctx context.Context, req *dto.GetWorkspacePermissionsReq) (*dto.GetWorkspacePermissionsResp, error) {
	workspaceUser, workspaceApp := req.User, req.App
	if req.ResourcePath != "" {
		_, parsedUser, parsedApp := permission.ParseFullCodePath(req.ResourcePath)
		if parsedUser == "" || parsedApp == "" {
			return nil, fmt.Errorf("resource_path 格式错误，无法解析 user 和 app: %s", req.ResourcePath)
		}
		workspaceUser, workspaceApp = parsedUser, parsedApp
	}
	if workspaceUser == "" || workspaceApp == "" {
		return nil, fmt.Errorf("必须提供 resource_path 或 user/app 参数")
	}

	// ⭐ 获取用户名：优先使用请求参数，否则从 context 获取（向后兼容）
	username := req.Username
	if username == "" {
		username = contextx.GetRequestUser(ctx)
		if username == "" {
			return nil, fmt.Errorf("无法获取用户信息（请提供 username 参数或确保 context 中包含用户信息）")
		}
	}

	// ⭐ 获取组织架构路径：优先使用请求参数，否则从 context 获取（向后兼容）
	deptPath := req.DepartmentFullPath
	if deptPath == "" {
		deptPath = contextx.GetRequestDepartmentFullPath(ctx)
	}

	// ⭐ 计算组织架构路径及其所有父级路径（用于日志记录）
	var deptPaths []string
	if deptPath != "" {
		deptPaths = s.getAllParentDeptPaths(deptPath)
		logger.Debugf(ctx, "[PermissionService] 查询权限: user=%s, deptPath=%s, parentPaths=%v",
			username, deptPath, deptPaths)
	} else {
		logger.Debugf(ctx, "[PermissionService] 用户无组织架构信息: user=%s，仅查询用户直接权限", username)
	}

	// ⭐ 直接使用 user 和 app，无需查询 app 表（性能优化）
	// ⭐ 注意：DepartmentPath 只需要传递当前路径，GetUserWorkspacePermissions 内部会重新计算所有父级路径
	// ⭐ 这样可以确保父级路径的计算逻辑统一（在 getUserRolePermissions 中处理）
	enterpriseReq := &enterprise.GetUserWorkspacePermissionsReq{
		User:           workspaceUser,
		App:            workspaceApp,
		Username:       username,
		DepartmentPath: deptPath, // ⭐ 只传递当前路径，父级路径在内部计算
	}

	enterpriseResp, err := s.permissionBackend().GetUserWorkspacePermissions(ctx, enterpriseReq)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 查询权限记录失败: resource_path=%s, user=%s, app=%s, username=%s, error=%v", req.ResourcePath, workspaceUser, workspaceApp, username, err)
		return nil, fmt.Errorf("查询权限记录失败: %w", err)
	}

	// ⭐ 转换为 DTO 格式
	records := make([]dto.PermissionRecord, 0, len(enterpriseResp.Records))
	for _, record := range enterpriseResp.Records {
		records = append(records, dto.PermissionRecord{
			ID:       0,  // 新系统不需要 ID
			User:     "", // 从 record.Resource 和 record.Action 中提取
			Resource: record.Resource,
			Action:   record.Action,
			AppID:    0, // 不再使用 AppID
		})
	}

	logger.Debugf(ctx, "[PermissionService] 查询权限成功: resource_path=%s, user=%s, app=%s, username=%s, total_records=%d", req.ResourcePath, workspaceUser, workspaceApp, username, len(records))

	// ⭐ 返回所有权限记录（包括用户权限和组织架构权限）
	return &dto.GetWorkspacePermissionsResp{
		Records: records,
	}, nil
}

// getAllParentDeptPaths 获取组织架构路径及其所有父级路径
// 例如：/org/master/bizit → [/org/master/bizit, /org/master, /org]
func (s *PermissionService) getAllParentDeptPaths(deptPath string) []string {
	if deptPath == "" {
		return []string{}
	}

	// 移除开头的斜杠
	path := strings.TrimPrefix(deptPath, "/")
	if path == "" {
		return []string{}
	}

	// 分割路径
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return []string{}
	}

	// 构建所有父级路径（包括自身）
	parentPaths := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		parentPath := "/" + strings.Join(parts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}

	return parentPaths
}
