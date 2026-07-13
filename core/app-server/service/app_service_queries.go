package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/gorm"
)

// GetApps 获取应用列表
func (a *AppService) GetApps(ctx context.Context, req *dto.GetAppsReq) (*dto.GetAppsResp, error) {
	// 设置分页默认值
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10 // 默认每页10条
	}

	// 从数据库获取应用列表（支持搜索和过滤）
	apps, total, err := a.appRepo.GetAppsWithPage(ctx, req.User, page, pageSize, req.Search, req.IncludeAll, req.Type == nil, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 转换为 AppInfo 列表
	appInfos := make([]*dto.AppInfo, 0, len(apps))
	for _, app := range apps {
		if !a.canReadApp(ctx, app, req.User) {
			continue
		}
		appInfos = append(appInfos, &dto.AppInfo{
			ID:                    app.ID,
			User:                  app.User,
			Code:                  app.Code,
			Name:                  app.Name,
			Status:                app.Status,
			Version:               app.Version,
			NatsID:                app.NatsID,
			HostID:                app.HostID,
			IsPublic:              app.IsPublic,
			IsPersonalWorkspace:   app.IsPersonalWorkspace,
			AccessMode:            string(model.NormalizeAppAccessMode(app.AccessMode)),
			HideUnauthorizedNodes: app.HideUnauthorizedNodes,
			Admins:                app.Admins,
			Type:                  int(app.Type),
			CreatedAt:             time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt:             time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		})
	}

	return &dto.GetAppsResp{
		PageInfoResp: dto.PageInfoResp{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: int(total),
			Items:      appInfos,
		},
	}, nil
}

func (a *AppService) mergeAccessibleApps(ctx context.Context, apps []*model.App, req *dto.GetAppsReq) ([]*model.App, error) {
	if a.teamAccess == nil || req == nil || req.User == "" || req.Type != nil {
		return apps, nil
	}
	grantedApps, err := a.teamAccess.ListAccessibleApps(ctx, req.User)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(apps)+len(grantedApps))
	merged := make([]*model.App, 0, len(apps)+len(grantedApps))
	matchesSearch := func(app *model.App) bool {
		if app == nil {
			return false
		}
		search := strings.ToLower(strings.TrimSpace(req.Search))
		if search == "" {
			return true
		}
		return strings.Contains(strings.ToLower(app.Name), search) || strings.Contains(strings.ToLower(app.Code), search)
	}
	for _, app := range apps {
		if app == nil {
			continue
		}
		key := app.User + "/" + app.Code
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, app)
	}
	for _, app := range grantedApps {
		if !matchesSearch(app) {
			continue
		}
		key := app.User + "/" + app.Code
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, app)
	}
	return merged, nil
}

func (a *AppService) canReadApp(ctx context.Context, app *model.App, currentUser string) bool {
	if app == nil {
		return false
	}
	if a.teamAccess == nil {
		return true
	}
	resourcePath := app.GetPrefix()
	ok, err := a.teamAccess.Can(ctx, app.User, app.Code, currentUser, resourcePath, access.ActionRead)
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	hasAnyAccess, err := a.teamAccess.HasAnyWorkspaceAccess(ctx, app.User, app.Code, currentUser)
	return err == nil && hasAnyAccess
}

// GetAppDetail 获取应用详情
func (a *AppService) GetAppDetail(ctx context.Context, req *dto.GetAppDetailReq) (*dto.GetAppDetailResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	app, err := a.appRepo.GetAppByUserNameContext(ctx, user, appCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", user, appCode)
		}
		return nil, fmt.Errorf("获取应用详情失败: %w", err)
	}

	return &dto.GetAppDetailResp{
		AppInfo: dto.AppInfo{
			ID:                    app.ID,
			User:                  app.User,
			Code:                  app.Code,
			Name:                  app.Name,
			Status:                app.Status,
			Version:               app.Version,
			NatsID:                app.NatsID,
			HostID:                app.HostID,
			IsPublic:              app.IsPublic,
			IsPersonalWorkspace:   app.IsPersonalWorkspace,
			AccessMode:            string(model.NormalizeAppAccessMode(app.AccessMode)),
			HideUnauthorizedNodes: app.HideUnauthorizedNodes,
			Admins:                app.Admins,
			Type:                  int(app.Type),
			CreatedAt:             time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt:             time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// GetAppByUserName 根据用户名和应用名获取应用信息
func (a *AppService) GetAppByUserName(ctx context.Context, user, app string) (*model.App, error) {
	return a.appRepo.GetAppByUserNameContext(ctx, user, app)
}
