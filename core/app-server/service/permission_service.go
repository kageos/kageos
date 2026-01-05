package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// PermissionService 权限管理服务
type PermissionService struct {
	permissionService enterprise.PermissionService
	casbinRuleRepo    *repository.CasbinRuleRepository // ⭐ 添加 CasbinRuleRepository，用于操作 casbin_rule 表
	appRepo           *repository.AppRepository        // ⭐ 添加 AppRepository，用于查询 app.id
}

// NewPermissionService 创建权限管理服务
func NewPermissionService(permissionService enterprise.PermissionService, casbinRuleRepo *repository.CasbinRuleRepository, appRepo *repository.AppRepository) *PermissionService {
	return &PermissionService{
		permissionService: permissionService,
		casbinRuleRepo:    casbinRuleRepo,
		appRepo:           appRepo,
	}
}

// AddPermission 添加权限
// ⭐ 支持两种类型的权限：
//  1. 用户权限：v0 = username（工作空间资源路径）
//  2. 组织架构权限：v0 = department_path（组织架构路径）
func (s *PermissionService) AddPermission(ctx context.Context, req *dto.AddPermissionReq) error {
	// ⭐ 判断是否是组织架构权限
	// 判断逻辑：
	//   1. 如果 Subject 是组织架构路径（以 /org 开头），则走组织架构权限逻辑
	//   2. 如果 ResourcePath 是组织架构路径（不以 /user/app 开头），则走组织架构权限逻辑
	//   3. 否则走用户权限逻辑
	isDeptPath := false
	if strings.HasPrefix(req.Subject, "/org") {
		// Subject 是组织架构路径
		isDeptPath = true
	} else {
		// 检查 ResourcePath 是否是组织架构路径
		isDeptPath = s.isDepartmentPath(ctx, req.ResourcePath)
	}

	if isDeptPath {
		// ⭐ 组织架构权限：v0 存储组织架构路径
		return s.addDepartmentPermission(ctx, req)
	}

	// ⭐ 工作空间权限：v0 存储用户名（原有逻辑）
	// 从 ResourcePath 解析出 user 和 app，查询 app.id
	appID, err := s.getAppIDFromResourcePath(ctx, req.ResourcePath)
	if err != nil {
		logger.Warnf(ctx, "[PermissionService] 获取 app.id 失败: resource=%s, error=%v，继续添加权限（不填充 app_id）",
			req.ResourcePath, err)
		// 如果获取 app.id 失败，记录警告但不中断流程（向后兼容）
		appID = 0
	}

	// ⭐ 如果是目录权限，同时添加精确路径和通配符路径的策略
	// 精确路径：用于匹配目录本身的权限（如 /task, directory:read）
	// 通配符路径：用于匹配子资源的权限（如 /task/*, directory:read）
	if strings.HasPrefix(req.Action, "directory:") || strings.HasPrefix(req.Action, "app:") {
		// 1. 先添加精确路径策略（用于匹配目录本身的权限）
		err := s.permissionService.AddPolicy(ctx, req.Subject, req.ResourcePath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加精确路径权限失败: subject=%s, resource=%s, action=%s, error=%v",
				req.Subject, req.ResourcePath, req.Action, err)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		// ⭐ 更新精确路径策略的 app_id（带重试机制，解决时序问题）
		if appID > 0 {
			if err := s.updateAppIDWithRetry(ctx, req.Subject, req.ResourcePath, req.Action, appID, "精确路径"); err != nil {
				logger.Warnf(ctx, "[PermissionService] 更新精确路径策略 app_id 失败: subject=%s, resource=%s, action=%s, app_id=%d, error=%v",
					req.Subject, req.ResourcePath, req.Action, appID, err)
				// 不中断流程，记录警告即可（可通过补偿脚本修复）
			}
		}

		// 2. 再添加通配符路径策略（用于匹配子资源的权限）
		wildcardPath := req.ResourcePath + "/*"
		err = s.permissionService.AddPolicy(ctx, req.Subject, wildcardPath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加通配符路径权限失败: subject=%s, resource=%s, action=%s, error=%v",
				req.Subject, wildcardPath, req.Action, err)
			// 如果通配符路径添加失败，尝试删除已添加的精确路径策略（回滚）
			_ = s.permissionService.RemovePolicy(ctx, req.Subject, req.ResourcePath, req.Action)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		// ⭐ 更新通配符路径策略的 app_id（带重试机制，解决时序问题）
		if appID > 0 {
			if err := s.updateAppIDWithRetry(ctx, req.Subject, wildcardPath, req.Action, appID, "通配符路径"); err != nil {
				logger.Warnf(ctx, "[PermissionService] 更新通配符路径策略 app_id 失败: subject=%s, resource=%s, action=%s, app_id=%d, error=%v",
					req.Subject, wildcardPath, req.Action, appID, err)
				// 不中断流程，记录警告即可（可通过补偿脚本修复）
			}
		}

		logger.Infof(ctx, "[PermissionService] 添加目录权限成功: subject=%s, resource=%s (exact=%s, wildcard=%s), action=%s, app_id=%d",
			req.Subject, req.ResourcePath, req.ResourcePath, wildcardPath, req.Action, appID)
		return nil
	}

	// 函数权限使用精确路径（因为函数是叶子节点，不需要通配符）
	err = s.permissionService.AddPolicy(ctx, req.Subject, req.ResourcePath, req.Action)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 添加权限失败: subject=%s, resource=%s, action=%s, error=%v",
			req.Subject, req.ResourcePath, req.Action, err)
		return fmt.Errorf("添加权限失败: %w", err)
	}

	// ⭐ 更新函数权限策略的 app_id（带重试机制，解决时序问题）
	if appID > 0 {
		if err := s.updateAppIDWithRetry(ctx, req.Subject, req.ResourcePath, req.Action, appID, "函数权限"); err != nil {
			logger.Warnf(ctx, "[PermissionService] 更新函数权限策略 app_id 失败: subject=%s, resource=%s, action=%s, app_id=%d, error=%v",
				req.Subject, req.ResourcePath, req.Action, appID, err)
			// 不中断流程，记录警告即可（可通过补偿脚本修复）
		}
	}

	logger.Infof(ctx, "[PermissionService] 添加权限成功: subject=%s, resource=%s, action=%s, app_id=%d",
		req.Subject, req.ResourcePath, req.Action, appID)
	return nil
}

// isDepartmentPath 判断是否是组织架构路径
// 组织架构路径格式：/org/master/bizit（不以 /user/app 开头）
// 工作空间路径格式：/user/app/...（至少包含 user 和 app 两部分）
func (s *PermissionService) isDepartmentPath(ctx context.Context, resourcePath string) bool {
	// 尝试解析为工作空间路径
	_, user, app, _ := permission.ParseFullCodePath(resourcePath)
	if user != "" && app != "" {
		// 能解析出 user 和 app，是工作空间路径
		return false
	}

	// 无法解析出 user 和 app，可能是组织架构路径
	// 组织架构路径通常以 / 开头，且不是 /user/app 格式
	if strings.HasPrefix(resourcePath, "/") && len(strings.Split(strings.Trim(resourcePath, "/"), "/")) > 0 {
		return true
	}

	return false
}

// addDepartmentPermission 添加组织架构权限
// ⭐ v0 存储组织架构路径（例如：/org/master/bizit）
// ⭐ 支持通配符路径，子部门自动继承权限
func (s *PermissionService) addDepartmentPermission(ctx context.Context, req *dto.AddPermissionReq) error {
	// ⭐ 确定组织架构路径：优先使用 Subject（如果 Subject 是组织架构路径），否则使用 ResourcePath
	var deptPath string
	var resourcePath string
	if strings.HasPrefix(req.Subject, "/org") {
		// Subject 是组织架构路径（例如：/org/sub/qsearch）
		// 这种情况下，给组织架构赋权，ResourcePath 是工作空间路径
		deptPath = req.Subject
		resourcePath = req.ResourcePath
	} else {
		// ResourcePath 是组织架构路径（例如：/org/master/bizit）
		// 这种情况下，给组织架构赋权，ResourcePath 就是组织架构路径
		deptPath = req.ResourcePath
		resourcePath = req.ResourcePath
	}

	// ⭐ 如果是目录权限，同时添加精确路径和通配符路径的策略
	if strings.HasPrefix(req.Action, "directory:") || strings.HasPrefix(req.Action, "app:") {
		// 1. 先添加精确路径策略（用于匹配部门本身的权限）
		err := s.permissionService.AddPolicy(ctx, deptPath, resourcePath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加组织架构精确路径权限失败: dept=%s, resource=%s, action=%s, error=%v",
				deptPath, resourcePath, req.Action, err)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		// 2. 再添加通配符路径策略（用于匹配子部门的权限）
		wildcardPath := resourcePath + "/*"
		err = s.permissionService.AddPolicy(ctx, deptPath, wildcardPath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加组织架构通配符路径权限失败: dept=%s, resource=%s, action=%s, error=%v",
				deptPath, wildcardPath, req.Action, err)
			// 如果通配符路径添加失败，尝试删除已添加的精确路径策略（回滚）
			_ = s.permissionService.RemovePolicy(ctx, deptPath, resourcePath, req.Action)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		logger.Infof(ctx, "[PermissionService] 添加组织架构权限成功: dept=%s, resource=%s (exact=%s, wildcard=%s), action=%s",
			deptPath, resourcePath, resourcePath, wildcardPath, req.Action)
		return nil
	}

	// 函数权限使用精确路径（因为函数是叶子节点，不需要通配符）
	err := s.permissionService.AddPolicy(ctx, deptPath, resourcePath, req.Action)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 添加组织架构权限失败: dept=%s, resource=%s, action=%s, error=%v",
			deptPath, resourcePath, req.Action, err)
		return fmt.Errorf("添加权限失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 添加组织架构权限成功: dept=%s, resource=%s, action=%s",
		deptPath, resourcePath, req.Action)
	return nil
}

// GetWorkspacePermissions 获取工作空间的所有权限
// ⭐ 优化：支持查询用户权限和组织架构权限（v0 可以是用户名或组织架构路径）
// ⭐ 一次性查询用户及其组织架构的所有权限，性能更好
// ⭐ 支持传递用户和组织架构参数，使方法可复用（既可以获取当前用户权限，也可以获取其他用户权限）
func (s *PermissionService) GetWorkspacePermissions(ctx context.Context, req *dto.GetWorkspacePermissionsReq) (*dto.GetWorkspacePermissionsResp, error) {
	// ⭐ 参数验证：必须提供 app_id
	if req.AppID <= 0 {
		return nil, fmt.Errorf("必须提供 app_id 参数")
	}

	// ⭐ 获取用户名：优先使用请求参数，否则从 context 获取（向后兼容）
	username := req.Username
	if username == "" {
		username = contextx.GetRequestUser(ctx)
		if username == "" {
			return nil, fmt.Errorf("无法获取用户信息（请提供 username 参数或确保 context 中包含用户信息）")
		}
	}

	// ⭐ 构建 v0 列表：用户名 + 组织架构路径 + 父级组织架构路径
	v0List := []string{username}

	// ⭐ 获取组织架构路径：优先使用请求参数，否则从 context 获取（向后兼容）
	deptPath := req.DepartmentFullPath
	if deptPath == "" {
		deptPath = contextx.GetRequestDepartmentFullPath(ctx)
	}

	if deptPath != "" {
		// 添加用户所属组织架构路径及其所有父级路径
		deptPaths := s.getAllParentDeptPaths(deptPath)
		v0List = append(v0List, deptPaths...)

		logger.Debugf(ctx, "[PermissionService] 查询权限: user=%s, deptPath=%s, parentPaths=%v, v0List=%v",
			username, deptPath, deptPaths, v0List)
	} else {
		logger.Debugf(ctx, "[PermissionService] 用户无组织架构信息: user=%s，仅查询用户直接权限", username)
	}

	// ⭐ 一次性查询所有权限（用户权限 + 组织架构权限）
	permissionRecords, err := s.casbinRuleRepo.GetPermissionsByAppIDAndV0List(req.AppID, v0List)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 查询权限记录失败: app_id=%d, v0List=%v, error=%v", req.AppID, v0List, err)
		return nil, fmt.Errorf("查询权限记录失败: %w", err)
	}

	// ⭐ 转换为 DTO 格式
	records := make([]dto.PermissionRecord, 0, len(permissionRecords))
	for _, record := range permissionRecords {
		records = append(records, dto.PermissionRecord{
			ID:       record.ID,
			User:     record.V0, // v0 可能是用户名或组织架构路径
			Resource: record.V1,
			Action:   record.V2,
			AppID:    record.AppID,
		})
	}

	logger.Debugf(ctx, "[PermissionService] 查询权限成功: app_id=%d, user=%s, total_records=%d", req.AppID, username, len(records))

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

// getAppIDFromResourcePath 从资源路径解析出 app.id
// 例如：/luobei/demo/docs → 查询 app where user='luobei' and code='demo' → 返回 app.id
func (s *PermissionService) getAppIDFromResourcePath(ctx context.Context, resourcePath string) (int64, error) {
	// 解析 full_code_path，提取 user 和 app
	_, user, app, _ := permission.ParseFullCodePath(resourcePath)
	if user == "" || app == "" {
		return 0, fmt.Errorf("无法从资源路径解析出 user 和 app: %s", resourcePath)
	}

	// 查询 app.id
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return 0, fmt.Errorf("查询应用失败: user=%s, app=%s, error=%w", user, app, err)
	}

	return appModel.ID, nil
}

// updateAppIDWithRetry 更新 app_id，带重试机制（解决时序/竞态问题）
// ⭐ 由于 AddPolicy 和 UpdateAppID 是两次独立的数据库操作，可能存在时序问题
// 如果第一次更新返回 0 行（记录还未写入），延迟后重试
func (s *PermissionService) updateAppIDWithRetry(ctx context.Context, username, resourcePath, action string, appID int64, logPrefix string) error {
	const maxRetries = 3
	const retryDelay = 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		rowsAffected, err := s.casbinRuleRepo.UpdateAppID(username, resourcePath, action, appID)
		if err != nil {
			return fmt.Errorf("更新 app_id 失败: %w", err)
		}

		// 如果更新成功（至少更新了 1 行），直接返回
		if rowsAffected > 0 {
			if attempt > 0 {
				logger.Infof(ctx, "[PermissionService] %s策略 app_id 更新成功（重试 %d 次）: user=%s, resource=%s, action=%s, app_id=%d",
					logPrefix, attempt, username, resourcePath, action, appID)
			}
			return nil
		}

		// 如果更新了 0 行，可能是记录还未写入，延迟后重试
		if attempt < maxRetries-1 {
			logger.Debugf(ctx, "[PermissionService] %s策略 app_id 更新返回 0 行，延迟 %v 后重试（第 %d 次）: user=%s, resource=%s, action=%s",
				logPrefix, retryDelay, attempt+1, username, resourcePath, action)
			time.Sleep(retryDelay)
		}
	}

	// 所有重试都失败，返回警告（不报错，可通过补偿脚本修复）
	logger.Warnf(ctx, "[PermissionService] %s策略 app_id 更新失败（重试 %d 次后仍返回 0 行）: user=%s, resource=%s, action=%s, app_id=%d，可能需要补偿脚本修复",
		logPrefix, maxRetries, username, resourcePath, action, appID)
	return nil // 不返回错误，记录警告即可
}
