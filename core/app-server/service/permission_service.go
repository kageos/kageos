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
func (s *PermissionService) AddPermission(ctx context.Context, req *dto.AddPermissionReq) error {
	// ⭐ 从 ResourcePath 解析出 user 和 app，查询 app.id
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
		err := s.permissionService.AddPolicy(ctx, req.Username, req.ResourcePath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加精确路径权限失败: user=%s, resource=%s, action=%s, error=%v",
				req.Username, req.ResourcePath, req.Action, err)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		// ⭐ 更新精确路径策略的 app_id（带重试机制，解决时序问题）
		if appID > 0 {
			if err := s.updateAppIDWithRetry(ctx, req.Username, req.ResourcePath, req.Action, appID, "精确路径"); err != nil {
				logger.Warnf(ctx, "[PermissionService] 更新精确路径策略 app_id 失败: user=%s, resource=%s, action=%s, app_id=%d, error=%v",
					req.Username, req.ResourcePath, req.Action, appID, err)
				// 不中断流程，记录警告即可（可通过补偿脚本修复）
			}
		}

		// 2. 再添加通配符路径策略（用于匹配子资源的权限）
		wildcardPath := req.ResourcePath + "/*"
		err = s.permissionService.AddPolicy(ctx, req.Username, wildcardPath, req.Action)
		if err != nil {
			logger.Errorf(ctx, "[PermissionService] 添加通配符路径权限失败: user=%s, resource=%s, action=%s, error=%v",
				req.Username, wildcardPath, req.Action, err)
			// 如果通配符路径添加失败，尝试删除已添加的精确路径策略（回滚）
			_ = s.permissionService.RemovePolicy(ctx, req.Username, req.ResourcePath, req.Action)
			return fmt.Errorf("添加权限失败: %w", err)
		}

		// ⭐ 更新通配符路径策略的 app_id（带重试机制，解决时序问题）
		if appID > 0 {
			if err := s.updateAppIDWithRetry(ctx, req.Username, wildcardPath, req.Action, appID, "通配符路径"); err != nil {
				logger.Warnf(ctx, "[PermissionService] 更新通配符路径策略 app_id 失败: user=%s, resource=%s, action=%s, app_id=%d, error=%v",
					req.Username, wildcardPath, req.Action, appID, err)
				// 不中断流程，记录警告即可（可通过补偿脚本修复）
			}
		}

		logger.Infof(ctx, "[PermissionService] 添加目录权限成功: user=%s, resource=%s (exact=%s, wildcard=%s), action=%s, app_id=%d",
			req.Username, req.ResourcePath, req.ResourcePath, wildcardPath, req.Action, appID)
		return nil
	}

	// 函数权限使用精确路径（因为函数是叶子节点，不需要通配符）
	err = s.permissionService.AddPolicy(ctx, req.Username, req.ResourcePath, req.Action)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 添加权限失败: user=%s, resource=%s, action=%s, error=%v",
			req.Username, req.ResourcePath, req.Action, err)
		return fmt.Errorf("添加权限失败: %w", err)
	}

	// ⭐ 更新函数权限策略的 app_id（带重试机制，解决时序问题）
	if appID > 0 {
		if err := s.updateAppIDWithRetry(ctx, req.Username, req.ResourcePath, req.Action, appID, "函数权限"); err != nil {
			logger.Warnf(ctx, "[PermissionService] 更新函数权限策略 app_id 失败: user=%s, resource=%s, action=%s, app_id=%d, error=%v",
				req.Username, req.ResourcePath, req.Action, appID, err)
			// 不中断流程，记录警告即可（可通过补偿脚本修复）
		}
	}

	logger.Infof(ctx, "[PermissionService] 添加权限成功: user=%s, resource=%s, action=%s, app_id=%d",
		req.Username, req.ResourcePath, req.Action, appID)
	return nil
}

// GetWorkspacePermissions 获取工作空间的所有权限
// ⭐ 直接通过 app_id + 用户信息查询权限记录并返回原始数据，让前端处理
func (s *PermissionService) GetWorkspacePermissions(ctx context.Context, req *dto.GetWorkspacePermissionsReq) (*dto.GetWorkspacePermissionsResp, error) {
	// ⭐ 参数验证：必须提供 app_id
	if req.AppID <= 0 {
		return nil, fmt.Errorf("必须提供 app_id 参数")
	}

	// ⭐ 从 context 中获取当前用户名（JWT 中间件已设置）
	username := contextx.GetRequestUser(ctx)
	if username == "" {
		return nil, fmt.Errorf("无法获取当前用户信息")
	}

	// ⭐ 直接通过 app_id + user 查询权限记录，返回原始数据
	permissionRecords, err := s.casbinRuleRepo.GetPermissionsByAppIDAndUser(req.AppID, username)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 查询权限记录失败: app_id=%d, user=%s, error=%v", req.AppID, username, err)
		return nil, fmt.Errorf("查询权限记录失败: %w", err)
	}

	// ⭐ 转换为 DTO 格式
	records := make([]dto.PermissionRecord, 0, len(permissionRecords))
	for _, record := range permissionRecords {
		records = append(records, dto.PermissionRecord{
			ID:       record.ID,
			User:     record.V0,
			Resource: record.V1,
			Action:   record.V2,
			AppID:    record.AppID,
		})
	}

	// ⭐ 直接返回原始权限记录，让前端处理
	return &dto.GetWorkspacePermissionsResp{
		Records: records,
	}, nil
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
