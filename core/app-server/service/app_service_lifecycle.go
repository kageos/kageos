package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

// CreateApp 创建应用
func (a *AppService) CreateApp(ctx context.Context, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	return a.createAppFlow(ctx, req)
}

// UpdateApp 更新应用（更新应用代码并重新编译部署）
func (a *AppService) UpdateApp(ctx context.Context, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	start := time.Now()
	user, appCode, app, err := a.resolveUpdateTargetApp(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] resolve target failed: resource_path=%s, err=%v, elapsed=%s",
			req.ResourcePath, err, time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	resolveElapsed := time.Since(start)

	// 调用 app-runtime 更新应用，使用应用所属的 HostID
	runtimeStart := time.Now()
	resp, err := a.appCall.UpdateApp(ctx, app.HostID, &dto.UpdateAppRuntimeReq{
		User:              user,
		App:               appCode,
		SourceFiles:       req.SourceFiles,
		Requirement:       req.Requirement,
		ChangeDescription: req.ChangeDescription,
		WriteOnly:         req.WriteOnly,
		ForceDiff:         req.ForceDiff,
	})
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] runtime update failed: user=%s, app=%s, hostID=%d, err=%v, resolveElapsed=%s, runtimeElapsed=%s, totalElapsed=%s",
			user, appCode, app.HostID, err,
			resolveElapsed.Truncate(time.Millisecond),
			time.Since(runtimeStart).Truncate(time.Millisecond),
			time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	runtimeElapsed := time.Since(runtimeStart)

	finalizeStart := time.Now()
	warnings, err := a.finalizeReleasedAppMetadata(ctx, "AppService:UpdateApp", app, user, appCode, resp.NewVersion, resp.Diff)
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] finalize metadata failed: user=%s, app=%s, newVersion=%s, err=%v, finalizeElapsed=%s, totalElapsed=%s",
			user, appCode, resp.NewVersion, err,
			time.Since(finalizeStart).Truncate(time.Millisecond),
			time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	resp.Warnings = append(resp.Warnings, warnings...)
	logger.Infof(ctx, "[AppService:UpdateApp] completed: user=%s, app=%s, oldVersion=%s, newVersion=%s, trace_id=%s, resolveElapsed=%s, runtimeElapsed=%s, finalizeElapsed=%s, totalElapsed=%s",
		user, appCode, resp.OldVersion, resp.NewVersion, updateAppTraceID(resp),
		resolveElapsed.Truncate(time.Millisecond),
		runtimeElapsed.Truncate(time.Millisecond),
		time.Since(finalizeStart).Truncate(time.Millisecond),
		time.Since(start).Truncate(time.Millisecond))

	return resp, nil
}

func updateAppTraceID(resp *dto.UpdateAppResp) string {
	if resp == nil || resp.BuildTrace == nil {
		return ""
	}
	return resp.BuildTrace.TraceID
}

func (a *AppService) resolveUpdateTargetApp(ctx context.Context, req *dto.UpdateAppReq) (string, string, *model.App, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return "", "", nil, err
	}

	app, err := a.appRepo.GetAppByUserNameContext(ctx, user, appCode)
	return user, appCode, app, err
}

func (a *AppService) persistReleasedAppVersion(user, appCode, newVersion string) error {
	if newVersion == "" {
		return nil
	}
	return a.appRepo.UpdateAppVersion(context.Background(), user, appCode, newVersion)
}

func (a *AppService) syncUpdatedAppMetadata(
	ctx context.Context,
	appID int64,
	diff *dto.DiffData,
) string {
	if diff == nil {
		return ""
	}

	if err := a.processAPIDiff(ctx, appID, diff); err != nil {
		return fmt.Sprintf("应用已发布，但函数元数据同步失败: %v", err)
	}

	return ""
}

func (a *AppService) finalizeReleasedAppMetadata(
	ctx context.Context,
	logPrefix string,
	app *model.App,
	user, appCode, newVersion string,
	diff *dto.DiffData,
) ([]string, error) {
	if app == nil {
		return nil, fmt.Errorf("应用不存在")
	}

	if err := a.persistReleasedAppVersion(user, appCode, newVersion); err != nil {
		return nil, err
	}

	warning := a.syncUpdatedAppMetadata(ctx, app.ID, diff)
	if warning == "" {
		return nil, nil
	}

	logger.Warnf(ctx, "[%s] %s user=%s app=%s newVersion=%s",
		logPrefix, warning, user, appCode, newVersion)
	return []string{warning}, nil
}

// DeleteApp 删除应用
func (a *AppService) DeleteApp(ctx context.Context, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}
	// 根据应用信息获取 NATS 连接
	app, err := a.appRepo.GetAppByUserNameContext(ctx, user, appCode)
	if err != nil {
		return nil, err
	}
	if app.IsPersonalWorkspace {
		return nil, fmt.Errorf("默认个人空间不支持删除。若要开始新的工作，可创建其他工作空间。")
	}

	// 调用 app-runtime 删除应用
	resp, err := a.appCall.DeleteApp(ctx, app.HostID, &dto.DeleteAppRuntimeReq{
		User: user,
		App:  appCode,
	})
	if err != nil {
		return nil, err
	}

	// 删除数据库记录
	err = a.appRepo.DeleteAppAndVersions(ctx, user, appCode)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateWorkspace 更新工作空间（只更新 MySQL 记录，不涉及容器更新）。
// 名称是展示字段：code、URL 和运行时目录保持不变。根目录会在同一事务中同步名称，
// 以避免工作空间列表与目录根节点显示不一致。
func (a *AppService) UpdateWorkspace(ctx context.Context, req *dto.UpdateWorkspaceReq) (*dto.UpdateWorkspaceResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	// 获取应用信息
	app, err := a.appRepo.GetAppByUserNameContext(ctx, user, appCode)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// app repository 会缓存模型指针；先复制一份，确保事务失败时不会污染缓存中的旧值。
	updatedApp := *app
	nameChanged := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("工作空间名称不能为空")
		}
		updatedApp.Name = name
		nameChanged = name != app.Name
		if nameChanged {
			exists, err := a.appRepo.ExistsAppNameForUser(ctx, user, name)
			if err != nil {
				return nil, fmt.Errorf("检查工作空间名称失败: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("工作空间名称已存在: %s", name)
			}
		}
	}
	if req.AccessMode != nil {
		accessMode := model.NormalizeAppAccessMode(model.AppAccessMode(*req.AccessMode))
		if !model.IsValidAppAccessMode(accessMode) {
			return nil, fmt.Errorf("工作空间访问模式无效: %s", *req.AccessMode)
		}
		updatedApp.AccessMode = accessMode
		if accessMode.IsOpenCollaboration() {
			updatedApp.IsPublic = true
		}
	}
	if req.Admins != nil {
		updatedApp.Admins = *req.Admins
	}
	if req.HideUnauthorizedNodes != nil {
		updatedApp.HideUnauthorizedNodes = *req.HideUnauthorizedNodes
	}

	if err := a.appRepo.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		if nameChanged {
			var rootNode model.ServiceTree
			if err := tx.Where("app_id = ? AND ref_id = ?", updatedApp.ID, updatedApp.ID).First(&rootNode).Error; err != nil {
				return fmt.Errorf("获取工作空间根目录失败: %w", err)
			}
			rootNode.Name = updatedApp.Name
			if err := tx.Save(&rootNode).Error; err != nil {
				return fmt.Errorf("更新工作空间根目录名称失败: %w", err)
			}
		}
		if err := tx.Save(&updatedApp).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("更新工作空间失败: %w", err)
	}
	a.appRepo.InvalidateAppCacheBoth(ctx, updatedApp.User, updatedApp.Code, updatedApp.ID)

	logger.Infof(ctx, "[AppService] 更新工作空间成功: user=%s, app=%s, name=%s, access_mode=%s, admins=%s, hide_unauthorized_nodes=%t",
		user, appCode, updatedApp.Name, updatedApp.AccessMode, updatedApp.Admins, updatedApp.HideUnauthorizedNodes)

	return &dto.UpdateWorkspaceResp{
		User:                  user,
		App:                   appCode,
		Name:                  updatedApp.Name,
		AccessMode:            string(model.NormalizeAppAccessMode(updatedApp.AccessMode)),
		Admins:                updatedApp.Admins,
		HideUnauthorizedNodes: updatedApp.HideUnauthorizedNodes,
	}, nil
}
