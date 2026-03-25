package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
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

	if err := validateGoPackageName(req.Code); err != nil {
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

	app := &model.App{
		Base: models.Base{
			CreatedBy: requestUser,
		},
		Version:           "v1",
		Code:              req.Code,
		Name:              req.Name,
		User:              tenantUser,
		NatsID:            selectedHost.NatsID,
		HostID:            selectedHost.ID,
		Status:            "enabled",
		IsPublic:          isPublic,
		Admins:            req.Admins,
		ShowOnlyPermitted: showOnlyPermitted,
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

// goPackageNameRegex 合法的 Go package 名称：以小写字母开头，后续可跟小写字母、数字、下划线。
var goPackageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// goKeywords Go 保留关键字，不能作为 package 名。
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// validateGoPackageName 校验字符串是否为合法的 Go package 名称。
// 规则：以小写字母开头，只能包含小写字母、数字、下划线，长度 2-50，不能是 Go 关键字。
func validateGoPackageName(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("工作空间英文标识不能为空")
	}
	if len(code) < 2 || len(code) > 50 {
		return fmt.Errorf("工作空间英文标识长度须为 2-50 个字符")
	}
	if !goPackageNameRegex.MatchString(code) {
		return fmt.Errorf("工作空间英文标识必须是合法的 Go package 名称：以小写字母开头，只能包含小写字母、数字和下划线")
	}
	if goKeywords[code] {
		return fmt.Errorf("工作空间英文标识不能使用 Go 保留关键字：%s", code)
	}
	return nil
}
