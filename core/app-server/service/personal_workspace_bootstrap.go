package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/gormx/models"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PersonalWorkspaceCode       = "home"
	legacyPersonalWorkspaceName = "我的空间"

	personalWorkspaceProvisioning = "provisioning"
	personalWorkspaceReady        = "ready"
	personalWorkspaceFailed       = "failed"
	personalWorkspaceStaleAfter   = 2 * time.Minute
)

var personalWorkspaceBootstrapGroup singleflight.Group

// BootstrapPersonalWorkspace 返回当前用户稳定可进入的工作空间：已有默认 Home 优先，
// 老用户沿用最早的已有空间，只有没有任何空间的新用户才创建私有 Home。
func (a *AppService) BootstrapPersonalWorkspace(ctx context.Context, user string) (*dto.BootstrapPersonalWorkspaceResp, error) {
	if a == nil || a.appRepo == nil || a.appCall == nil {
		return nil, fmt.Errorf("个人空间服务未初始化")
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return nil, fmt.Errorf("无法获取用户信息")
	}

	result, err, _ := personalWorkspaceBootstrapGroup.Do(user, func() (interface{}, error) {
		return a.bootstrapPersonalWorkspace(ctx, user)
	})
	if err != nil {
		return nil, err
	}
	resp, ok := result.(*dto.BootstrapPersonalWorkspaceResp)
	if !ok || resp == nil {
		return nil, fmt.Errorf("个人空间初始化返回异常")
	}
	return resp, nil
}

func (a *AppService) bootstrapPersonalWorkspace(ctx context.Context, user string) (*dto.BootstrapPersonalWorkspaceResp, error) {
	if app, found, err := a.findBootstrapWorkspace(ctx, user); err != nil {
		return nil, err
	} else if found {
		if err := a.ensurePersonalWorkspaceName(ctx, app); err != nil {
			return nil, err
		}
		return personalWorkspaceResponse(app, false), nil
	}

	claimed, err := a.claimPersonalWorkspaceBootstrap(ctx, user)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return a.waitForPersonalWorkspace(ctx, user)
	}

	selectedHost, err := a.selectHostForPersonalWorkspace(ctx)
	if err != nil {
		a.failPersonalWorkspaceBootstrap(ctx, user, err)
		return nil, err
	}
	private := false
	req := &dto.CreateAppReq{
		User:     user,
		Code:     PersonalWorkspaceCode,
		Name:     personalWorkspaceName(user),
		IsPublic: &private,
	}
	if _, err := a.provisionRuntimeApp(ctx, selectedHost.ID, req); err != nil {
		err = fmt.Errorf("创建默认个人空间运行环境失败: %w", err)
		a.failPersonalWorkspaceBootstrap(ctx, user, err)
		return nil, err
	}

	app, root := a.buildInitialAppAndRoot(user, user, req, selectedHost)
	app.IsPersonalWorkspace = true
	if err := a.persistCreatedApp(ctx, app, root); err != nil {
		err = fmt.Errorf("保存默认个人空间失败: %w", err)
		a.failPersonalWorkspaceBootstrap(ctx, user, err)
		return nil, err
	}
	if err := a.appRepo.GetDB().WithContext(ctx).Model(&model.PersonalWorkspaceBootstrap{}).
		Where("user = ?", user).
		Updates(map[string]interface{}{"status": personalWorkspaceReady, "app_id": app.ID, "error": ""}).Error; err != nil {
		// App 和根节点已经成功落库；下次请求能直接发现它，不因状态记录失败误报创建失败。
		return personalWorkspaceResponse(app, true), nil
	}
	return personalWorkspaceResponse(app, true), nil
}

func personalWorkspaceName(tenantUser string) string {
	name := strings.TrimSpace(tenantUser)
	if name == "" {
		return "默认空间"
	}
	return name + " 的默认空间"
}

// ensurePersonalWorkspaceName 只迁移平台曾自动生成的旧名称，用户自定义名称保持不变。
func (a *AppService) ensurePersonalWorkspaceName(ctx context.Context, app *model.App) error {
	if app == nil || !app.IsPersonalWorkspace || strings.TrimSpace(app.Name) != legacyPersonalWorkspaceName {
		return nil
	}
	name := personalWorkspaceName(app.User)
	db := a.appRepo.GetDB().WithContext(ctx)
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.App{}).Where("id = ?", app.ID).Update("name", name).Error; err != nil {
			return err
		}
		return tx.Model(&model.ServiceTree{}).
			Where("app_id = ? AND full_code_path = ?", app.ID, app.GetPrefix()).
			Update("name", name).Error
	}); err != nil {
		return fmt.Errorf("更新默认空间名称失败: %w", err)
	}
	app.Name = name
	a.appRepo.InvalidateAppCacheBoth(app.User, app.Code, app.ID)
	return nil
}

func (a *AppService) findBootstrapWorkspace(ctx context.Context, user string) (*model.App, bool, error) {
	db := a.appRepo.GetDB().WithContext(ctx)
	var home model.App
	err := db.Where("user = ? AND code = ? AND is_personal_workspace = ?", user, PersonalWorkspaceCode, true).First(&home).Error
	if err == nil {
		return &home, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("查询默认个人空间失败: %w", err)
	}

	var existing model.App
	err = db.Where("user = ? AND type = ?", user, model.AppTypeUser).
		Order("created_at ASC, id ASC").First(&existing).Error
	if err == nil {
		return &existing, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("查询已有工作空间失败: %w", err)
}

func (a *AppService) claimPersonalWorkspaceBootstrap(ctx context.Context, user string) (bool, error) {
	claimed := false
	err := a.appRepo.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.PersonalWorkspaceBootstrap
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user = ?", user).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = model.PersonalWorkspaceBootstrap{
				Base:   models.Base{CreatedBy: user, UpdatedBy: user},
				User:   user,
				Status: personalWorkspaceProvisioning,
			}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
			if result.Error != nil {
				return fmt.Errorf("占用个人空间初始化任务失败: %w", result.Error)
			}
			// 唯一键竞争时 RowsAffected=0，由另一个实例继续创建，本请求进入等待。
			claimed = result.RowsAffected == 1
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取个人空间初始化状态失败: %w", err)
		}

		updatedAt := time.Time(row.UpdatedAt)
		if row.Status == personalWorkspaceProvisioning && time.Since(updatedAt) < personalWorkspaceStaleAfter {
			return nil
		}
		claimed = true
		return tx.Model(&row).Updates(map[string]interface{}{
			"status": personalWorkspaceProvisioning,
			"app_id": 0,
			"error":  "",
		}).Error
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

func (a *AppService) waitForPersonalWorkspace(ctx context.Context, user string) (*dto.BootstrapPersonalWorkspaceResp, error) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("个人空间仍在创建中，请稍后重试")
		case <-ticker.C:
			if app, found, err := a.findBootstrapWorkspace(ctx, user); err != nil {
				return nil, err
			} else if found {
				return personalWorkspaceResponse(app, false), nil
			}
			var row model.PersonalWorkspaceBootstrap
			err := a.appRepo.GetDB().WithContext(ctx).Where("user = ?", user).First(&row).Error
			if err == nil && row.Status == personalWorkspaceFailed {
				return nil, fmt.Errorf("创建默认个人空间失败: %s", row.Error)
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
	}
}

func (a *AppService) failPersonalWorkspaceBootstrap(ctx context.Context, user string, cause error) {
	if a == nil || a.appRepo == nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	_ = a.appRepo.GetDB().WithContext(writeCtx).Model(&model.PersonalWorkspaceBootstrap{}).
		Where("user = ?", user).
		Updates(map[string]interface{}{"status": personalWorkspaceFailed, "error": cause.Error()}).Error
}

func (a *AppService) selectHostForPersonalWorkspace(ctx context.Context) (*model.Host, error) {
	var hosts []*model.Host
	if err := a.appRepo.GetDB().WithContext(ctx).Find(&hosts).Error; err != nil {
		return nil, fmt.Errorf("无法获取可用的主机: %w", err)
	}
	var selected *model.Host
	for _, host := range hosts {
		if host.Status == "enabled" && (selected == nil || host.AppCount < selected.AppCount) {
			selected = host
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("没有可用的主机")
	}
	return selected, nil
}

func personalWorkspaceResponse(app *model.App, created bool) *dto.BootstrapPersonalWorkspaceResp {
	return &dto.BootstrapPersonalWorkspaceResp{App: appInfoFromModel(app), Created: created}
}

func appInfoFromModel(app *model.App) dto.AppInfo {
	return dto.AppInfo{
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
		HideUnauthorizedNodes: app.HideUnauthorizedNodes,
		Admins:                app.Admins,
		Type:                  int(app.Type),
		CreatedAt:             time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt:             time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
	}
}
