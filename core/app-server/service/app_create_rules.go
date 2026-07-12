package service

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/gormx/models"
	"github.com/kageos/kageos/pkg/naming"
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
	if req.Code == PersonalWorkspaceCode {
		return "", "", fmt.Errorf("工作空间英文标识「%s」为系统保留，请换一个", PersonalWorkspaceCode)
	}

	if exists, err := a.appRepo.ExistsAppNameForUser(tenantUser, req.Name); err != nil {
		return "", "", fmt.Errorf("检查应用名称唯一性失败: %w", err)
	} else if exists {
		return "", "", fmt.Errorf("应用名称已存在: %s", req.Name)
	}

	accessMode := model.NormalizeAppAccessMode(model.AppAccessMode(req.AccessMode))
	if !model.IsValidAppAccessMode(accessMode) {
		return "", "", fmt.Errorf("工作空间访问模式无效: %s", req.AccessMode)
	}
	req.AccessMode = string(accessMode)

	return requestUser, tenantUser, nil
}

func (a *AppService) buildInitialAppAndRoot(requestUser, tenantUser string, req *dto.CreateAppReq, selectedHost *model.Host) (*model.App, *model.ServiceTree) {
	accessMode := model.NormalizeAppAccessMode(model.AppAccessMode(req.AccessMode))
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	if accessMode.IsOpenCollaboration() {
		isPublic = true
	}
	hideUnauthorizedNodes := false
	if req.HideUnauthorizedNodes != nil {
		hideUnauthorizedNodes = *req.HideUnauthorizedNodes
	}

	app := &model.App{
		Base: models.Base{
			CreatedBy: requestUser,
		},
		Version:               "v1",
		Code:                  req.Code,
		Name:                  req.Name,
		User:                  tenantUser,
		NatsID:                selectedHost.NatsID,
		HostID:                selectedHost.ID,
		Status:                "enabled",
		IsPublic:              isPublic,
		AccessMode:            accessMode,
		HideUnauthorizedNodes: hideUnauthorizedNodes,
		Admins:                req.Admins,
	}

	rootNode := &model.ServiceTree{
		Name:         req.Name,
		Code:         req.Code,
		Type:         model.ServiceTreeTypePackage,
		Admins:       req.Admins,
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
	if err := a.appRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := createAppRecord(tx, app); err != nil {
			return err
		}

		rootNode.AppID = app.ID
		rootNode.RefID = app.ID
		if err := tx.Create(rootNode).Error; err != nil {
			logger.Errorf(ctx, "[AppService] 创建 service_tree 根节点失败: app_id=%d, error=%v", app.ID, err)
			return fmt.Errorf("创建工作空间根节点失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	a.appRepo.InvalidateAppCacheBoth(app.User, app.Code, app.ID)

	logger.Infof(ctx, "[AppService] 创建 service_tree 根节点成功: app_id=%d, root_id=%d, full_code_path=%s",
		app.ID, rootNode.ID, rootNode.FullCodePath)
	return nil
}

// createAppRecord preserves an explicit private choice. App.IsPublic keeps a
// database default of true for historical callers; GORM therefore omits a
// false zero-value during Create and lets the database default overwrite it.
// Correct it in the same transaction so private workspaces are never visible
// outside the transaction as public.
func createAppRecord(tx *gorm.DB, app *model.App) error {
	requestedPublic := app.IsPublic
	if err := tx.Create(app).Error; err != nil {
		return err
	}
	if requestedPublic {
		return nil
	}
	if err := tx.Model(&model.App{}).Where("id = ?", app.ID).Update("is_public", false).Error; err != nil {
		return fmt.Errorf("设置工作空间为私有失败: %w", err)
	}
	app.IsPublic = false
	return nil
}
