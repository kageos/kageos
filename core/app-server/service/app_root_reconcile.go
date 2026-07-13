package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

type AppRootReconcileResult struct {
	Checked int
	Created int
	Updated int
}

func ReconcileAppRootServiceTrees(
	ctx context.Context,
	appRepo *repository.AppRepository,
	serviceTreeRepo *repository.ServiceTreeRepository,
) (AppRootReconcileResult, error) {
	var result AppRootReconcileResult
	if appRepo == nil || serviceTreeRepo == nil {
		return result, fmt.Errorf("app root reconcile requires appRepo and serviceTreeRepo")
	}

	apps, err := appRepo.GetAllApps(ctx)
	if err != nil {
		return result, fmt.Errorf("查询应用列表失败: %w", err)
	}

	for _, appModel := range apps {
		if appModel == nil {
			continue
		}
		user := strings.TrimSpace(appModel.User)
		appCode := strings.TrimSpace(appModel.Code)
		if user == "" || appCode == "" {
			logger.Warnf(ctx, "[AppRootReconcile] 跳过无效应用: id=%d user=%q code=%q", appModel.ID, appModel.User, appModel.Code)
			continue
		}
		result.Checked++

		rootPath := "/" + user + "/" + appCode
		root, err := serviceTreeRepo.GetServiceTreeByFullPath(ctx, rootPath)
		if err == nil && root != nil {
			if normalizeAppRootServiceTree(root, appModel) {
				if err := serviceTreeRepo.UpdateServiceTree(ctx, root); err != nil {
					return result, fmt.Errorf("更新应用根节点失败 path=%s: %w", rootPath, err)
				}
				result.Updated++
			}
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return result, fmt.Errorf("查询应用根节点失败 path=%s: %w", rootPath, err)
		}

		root = newAppRootServiceTree(appModel, rootPath)
		if err := serviceTreeRepo.Create(ctx, root); err != nil {
			return result, fmt.Errorf("创建应用根节点失败 path=%s: %w", rootPath, err)
		}
		result.Created++
	}

	if result.Created > 0 || result.Updated > 0 {
		logger.Infof(ctx, "[AppRootReconcile] 应用根节点已协调: checked=%d created=%d updated=%d",
			result.Checked, result.Created, result.Updated)
	} else {
		logger.Infof(ctx, "[AppRootReconcile] 应用根节点无需修复: checked=%d", result.Checked)
	}
	return result, nil
}

func newAppRootServiceTree(appModel *model.App, rootPath string) *model.ServiceTree {
	root := &model.ServiceTree{
		Name:         appRootFirstNonEmptyString(strings.TrimSpace(appModel.Name), strings.TrimSpace(appModel.Code)),
		Code:         strings.TrimSpace(appModel.Code),
		Type:         model.ServiceTreeTypePackage,
		Admins:       strings.TrimSpace(appModel.Admins),
		AppID:        appModel.ID,
		RefID:        appModel.ID,
		FullCodePath: rootPath,
		Version:      strings.TrimSpace(appModel.Version),
		VersionNum:   appModel.GetVersionNumber(),
	}
	actor := appRootFirstNonEmptyString(strings.TrimSpace(appModel.CreatedBy), strings.TrimSpace(appModel.User))
	root.CreatedBy = actor
	root.UpdatedBy = actor
	return root
}

func normalizeAppRootServiceTree(root *model.ServiceTree, appModel *model.App) bool {
	changed := false
	if root.AppID != appModel.ID {
		root.AppID = appModel.ID
		changed = true
	}
	if root.RefID != appModel.ID {
		root.RefID = appModel.ID
		changed = true
	}
	if root.Type != model.ServiceTreeTypePackage {
		root.Type = model.ServiceTreeTypePackage
		changed = true
	}
	if root.Code != strings.TrimSpace(appModel.Code) {
		root.Code = strings.TrimSpace(appModel.Code)
		changed = true
	}
	if strings.TrimSpace(root.Name) == "" {
		root.Name = appRootFirstNonEmptyString(strings.TrimSpace(appModel.Name), strings.TrimSpace(appModel.Code))
		changed = true
	}
	if strings.TrimSpace(root.Admins) == "" && strings.TrimSpace(appModel.Admins) != "" {
		root.Admins = strings.TrimSpace(appModel.Admins)
		changed = true
	}
	if strings.TrimSpace(root.Version) == "" && strings.TrimSpace(appModel.Version) != "" {
		root.Version = strings.TrimSpace(appModel.Version)
		changed = true
	}
	if root.VersionNum == 0 && appModel.GetVersionNumber() != 0 {
		root.VersionNum = appModel.GetVersionNumber()
		changed = true
	}
	return changed
}

func appRootFirstNonEmptyString(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}
