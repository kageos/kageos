package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// PersonalWorkspaceCode is intentionally stable. It is used in URLs and
	// must not change when the user later renames the workspace display name.
	PersonalWorkspaceCode = "home"
	PersonalWorkspaceName = "我的空间"
)

// personalWorkspaceBootstrapGroup coalesces duplicate bootstrap requests from
// the same browser/session (for example, two tabs opening at once). A database
// row lock in bootstrapPersonalWorkspace then covers concurrent app-server
// processes sharing the same control-plane database.
var personalWorkspaceBootstrapGroup singleflight.Group

// BootstrapPersonalWorkspace returns a deterministic workspace for the current
// user without manufacturing a new home for existing users:
//
//  1. return /{user}/home when the system-provisioned personal-home marker exists;
//  2. otherwise return the user's earliest existing user workspace;
//  3. otherwise create a private /{user}/home named "我的空间".
//
// The caller must obtain user from verified JWT identity; it is deliberately
// not accepted from an HTTP request body.
func (a *AppService) BootstrapPersonalWorkspace(ctx context.Context, user string) (*dto.BootstrapPersonalWorkspaceResp, error) {
	if a == nil || a.appRepo == nil {
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
	if app, found, err := a.findBootstrapWorkspace(user); err != nil {
		return nil, err
	} else if found {
		return bootstrapPersonalWorkspaceResp(app, false), nil
	}

	var resp *dto.BootstrapPersonalWorkspaceResp
	err := a.appRepo.GetDB().Transaction(func(tx *gorm.DB) error {
		// App has no natural-key database constraint because it is soft-deleted.
		// Locking one durable Host row gives this first-use operation a small,
		// cross-process critical section without changing normal app deletion and
		// recreation semantics. Bootstrap is rare, so global serialization here is
		// preferable to introducing a new ownership table in P0.
		if err := lockPersonalWorkspaceBootstrap(tx); err != nil {
			return err
		}

		if existing, found, err := findBootstrapWorkspaceTx(tx, user); err != nil {
			return err
		} else if found {
			resp = bootstrapPersonalWorkspaceResp(existing, false)
			return nil
		}

		selectedHost, err := selectHostForCreateAppTx(tx)
		if err != nil {
			return err
		}

		private := false
		req := &dto.CreateAppReq{
			User:     user,
			Code:     PersonalWorkspaceCode,
			Name:     PersonalWorkspaceName,
			IsPublic: &private,
		}
		if _, err := a.provisionRuntimeApp(ctx, selectedHost.ID, req); err != nil {
			return fmt.Errorf("创建默认个人空间运行环境失败: %w", err)
		}

		app, rootNode := a.buildInitialAppAndRoot(user, user, req, selectedHost)
		app.IsPersonalWorkspace = true
		if err := createAppRecord(tx, app); err != nil {
			return fmt.Errorf("保存默认个人空间失败: %w", err)
		}
		rootNode.AppID = app.ID
		rootNode.RefID = app.ID
		if err := tx.Create(rootNode).Error; err != nil {
			return fmt.Errorf("创建默认个人空间根节点失败: %w", err)
		}

		resp = bootstrapPersonalWorkspaceResp(app, true)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("个人空间初始化返回为空")
	}

	// The normal repository cache is intentionally bypassed while holding the
	// transaction lock, so invalidate both lookup keys after a successful create.
	if resp.Created {
		a.appRepo.InvalidateAppCacheBoth(resp.App.User, resp.App.Code, resp.App.ID)
	}
	return resp, nil
}

func (a *AppService) findBootstrapWorkspace(user string) (*model.App, bool, error) {
	if app, found, err := a.findExistingPersonalHome(user); err != nil || found {
		return app, found, err
	}

	app, err := a.appRepo.GetFirstUserApp(user)
	if err == nil {
		return app, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("查询已有工作空间失败: %w", err)
}

func (a *AppService) findExistingPersonalHome(user string) (*model.App, bool, error) {
	app, err := a.appRepo.GetAppByUserName(user, PersonalWorkspaceCode)
	if err == nil {
		return app, app.IsPersonalWorkspace, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("查询默认个人空间失败: %w", err)
}

func lockPersonalWorkspaceBootstrap(tx *gorm.DB) error {
	var lockHost model.Host
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC").First(&lockHost).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("没有可用的主机")
		}
		return fmt.Errorf("锁定个人空间初始化失败: %w", err)
	}
	return nil
}

func findBootstrapWorkspaceTx(tx *gorm.DB, user string) (*model.App, bool, error) {
	var home model.App
	err := tx.Where("user = ? AND code = ?", user, PersonalWorkspaceCode).First(&home).Error
	if err == nil {
		if home.IsPersonalWorkspace {
			return &home, true, nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("查询默认个人空间失败: %w", err)
	}

	var existing model.App
	err = tx.
		Where("user = ? AND type = ?", user, model.AppTypeUser).
		Order("created_at ASC, id ASC").
		First(&existing).Error
	if err == nil {
		return &existing, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("查询已有工作空间失败: %w", err)
}

func selectHostForCreateAppTx(tx *gorm.DB) (*model.Host, error) {
	var hosts []*model.Host
	if err := tx.Model(&model.Host{}).Find(&hosts).Error; err != nil {
		return nil, fmt.Errorf("无法获取可用的主机: %w", err)
	}

	var selectedHost *model.Host
	for _, host := range hosts {
		if host.Status != "enabled" {
			continue
		}
		if selectedHost == nil || host.AppCount < selectedHost.AppCount {
			selectedHost = host
		}
	}
	if selectedHost == nil {
		return nil, fmt.Errorf("没有可用的主机")
	}
	return selectedHost, nil
}

func bootstrapPersonalWorkspaceResp(app *model.App, created bool) *dto.BootstrapPersonalWorkspaceResp {
	return &dto.BootstrapPersonalWorkspaceResp{
		App: dto.AppInfo{
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
		Created: created,
	}
}
