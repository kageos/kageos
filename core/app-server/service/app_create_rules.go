package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
)

func (a *AppService) validateCreateAppRequest(ctx context.Context, req *dto.CreateAppReq) (string, string, error) {
	tenantUser := req.User
	if tenantUser == "" {
		return "", "", fmt.Errorf("租户用户信息不能为空")
	}

	requestUser := contextx.GetRequestUser(ctx)
	if requestUser == "" {
		return "", "", fmt.Errorf("请求用户信息不能为空")
	}

	appCount, err := a.appRepo.CountApps()
	if err != nil {
		logger.Warnf(ctx, "[AppService] Failed to count apps: %v", err)
	} else {
		licenseMgr := license.GetManager()
		if err := licenseMgr.CheckAppLimit(int(appCount)); err != nil {
			return "", "", err
		}
	}

	// system 是内置用户，跳过远程验证，避免系统初始化时的循环依赖。
	if tenantUser != SystemUsername {
		if _, err := apicall.GetUserByUsername(ctx, &dto.QueryUserReq{Username: tenantUser}); err != nil {
			return "", "", fmt.Errorf("租户用户 %s 不存在: %w", tenantUser, err)
		}
	}

	req.Code = naming.NormalizeGoPackageName(req.Code)
	if err := naming.ValidateGoPackageNameLength(req.Code, "工作空间英文标识", 2, naming.MaxGoPackageNameLength); err != nil {
		return "", "", err
	}

	if exists, err := a.appRepo.ExistsAppNameForUser(tenantUser, req.Name); err != nil {
		return "", "", fmt.Errorf("检查应用名称唯一性失败: %w", err)
	} else if exists {
		return "", "", fmt.Errorf("应用名称已存在: %s", req.Name)
	}

	return requestUser, tenantUser, nil
}

func (a *AppService) buildInitialAppAndRoot(requestUser, tenantUser string, req *dto.CreateAppReq, selectedHost *model.Host) (*model.App, *model.ServiceTree) {
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	showOnlyPermitted := false
	if req.ShowOnlyPermitted != nil {
		showOnlyPermitted = *req.ShowOnlyPermitted
	}

	permissionEnforced := DefaultWorkspacePermissionEnforced()
	if req.PermissionEnforced != nil {
		permissionEnforced = *req.PermissionEnforced
	}

	app := &model.App{
		Base: models.Base{
			CreatedBy: requestUser,
		},
		Version:            "v1",
		Code:               req.Code,
		Name:               req.Name,
		User:               tenantUser,
		NatsID:             selectedHost.NatsID,
		HostID:             selectedHost.ID,
		Status:             "enabled",
		IsPublic:           isPublic,
		Admins:             req.Admins,
		ShowOnlyPermitted:  showOnlyPermitted,
		PermissionEnforced: permissionEnforced,
	}

	rootNode := &model.ServiceTree{
		Name:         req.Name,
		Code:         req.Code,
		Type:         model.ServiceTreeTypePackage,
		Admins:       req.Admins,
		PendingCount: 0,
		FullCodePath: fmt.Sprintf("/%s/%s", tenantUser, req.Code),
		Version:      "v1",
		VersionNum:   1,
		Base: models.Base{
			CreatedBy: requestUser,
			UpdatedBy: requestUser,
		},
	}

	return app, rootNode
}

func (a *AppService) persistCreatedApp(ctx context.Context, app *model.App, rootNode *model.ServiceTree) error {
	if err := a.appRepo.CreateApp(app); err != nil {
		return err
	}

	rootNode.AppID = app.ID
	rootNode.RefID = app.ID
	if err := a.serviceTreeRepo.Create(rootNode); err != nil {
		logger.Errorf(ctx, "[AppService] 创建 service_tree 根节点失败: app_id=%d, error=%v", app.ID, err)
		// 根节点创建失败会导致服务树无法显示，暂时保持现有行为：返回错误，不做回滚。
		return fmt.Errorf("创建工作空间根节点失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 创建 service_tree 根节点成功: app_id=%d, root_id=%d, full_code_path=%s",
		app.ID, rootNode.ID, rootNode.FullCodePath)
	return nil
}
