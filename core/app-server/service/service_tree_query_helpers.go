package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func getServiceTreeByAppModelImpl(s *ServiceTreeService, ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	var trees []*model.ServiceTree
	var err error
	if nodeType != "" {
		trees, err = s.serviceTreeRepo.BuildServiceTreeByType(appModel.ID, nodeType)
	} else {
		trees, err = s.serviceTreeRepo.BuildServiceTree(appModel.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build service tree: %w", err)
	}

	if len(trees) == 0 {
		return nil, fmt.Errorf("服务树为空，app_id=%d，请检查根节点是否已创建", appModel.ID)
	}

	if !trees[0].IsRoot() || trees[0].RefID != appModel.ID {
		return nil, fmt.Errorf("根节点无效，app_id=%d, full_code_path=%s, ref_id=%d", appModel.ID, trees[0].FullCodePath, trees[0].RefID)
	}

	rootNode := trees[0]
	logger.Debugf(ctx, "[ServiceTreeService] 获取服务树成功: root_id=%d, app_id=%d, full_code_path=%s, children_count=%d",
		rootNode.ID, appModel.ID, rootNode.FullCodePath, len(rootNode.Children))

	var permissionsMap map[string]map[string]bool
	var isAdmin bool
	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && appModel.ID > 0 && s.permissionService != nil {
		if isWorkspaceAdmin(username, appModel.Admins) {
			isAdmin = true
			logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，设置 isAdmin=true", username)
		}

		permsMap, err := s.calculatePermissions(ctx, appModel.User, appModel.Code, trees, appModel.Admins, username)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 计算权限失败: app_id=%d, error=%v，继续返回服务树（无权限信息）", appModel.ID, err)
		} else {
			permissionsMap = permsMap
			logger.Debugf(ctx, "[ServiceTreeService] 权限计算完成: app_id=%d, username=%s, isAdmin=%v", appModel.ID, username, isAdmin)
		}
	}

	rootResp := s.convertToGetServiceTreeResp(ctx, rootNode, permissionsMap, isAdmin)

	if appModel.ShowOnlyPermitted && !isAdmin && permissionsMap != nil {
		rootResp = s.filterTreeByPermission(rootResp)
		logger.Debugf(ctx, "[ServiceTreeService] 已按权限过滤服务树: app_id=%d, username=%s", appModel.ID, username)
	}

	logger.Debugf(ctx, "[ServiceTreeService] 服务树转换完成: root_id=%d, children_count=%d",
		rootNode.ID, len(rootResp.Children))

	return []*dto.GetServiceTreeResp{rootResp}, nil
}

func getAppWithServiceTreeImpl(s *ServiceTreeService, ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	appModel, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", req.User, req.App)
		}
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	appInfo := dto.AppInfo{
		ID:                appModel.ID,
		User:              appModel.User,
		Code:              appModel.Code,
		Name:              appModel.Name,
		Status:            appModel.Status,
		Version:           appModel.Version,
		NatsID:            appModel.NatsID,
		HostID:            appModel.HostID,
		IsPublic:          appModel.IsPublic,
		Admins:            appModel.Admins,
		Type:              int(appModel.Type),
		ShowOnlyPermitted: appModel.ShowOnlyPermitted,
		CreatedAt:         time.Time(appModel.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt:         time.Time(appModel.UpdatedAt).Format("2006-01-02 15:04:05"),
	}

	serviceTreeResp, err := s.getServiceTreeByAppModel(ctx, appModel, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录树失败: %w", err)
	}

	expandedKeys := s.calculateExpandedKeys(serviceTreeResp)

	return &dto.GetAppWithServiceTreeResp{
		App:          appInfo,
		ServiceTree:  serviceTreeResp,
		ExpandedKeys: expandedKeys,
	}, nil
}

func getServiceTreeDetailImpl(s *ServiceTreeService, ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	var tree *model.ServiceTree
	var err error

	if req.ID > 0 {
		tree, err = s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	} else {
		return nil, fmt.Errorf("必须提供 ID 或 full_code_path")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("获取服务目录失败: %w", err)
	}

	resp := &dto.GetServiceTreeDetailResp{
		ID:              tree.ID,
		Name:            tree.Name,
		Code:            tree.Code,
		Type:            tree.Type,
		Description:     tree.Description,
		Tags:            tree.Tags,
		AppID:           tree.AppID,
		RefID:           tree.RefID,
		FullCodePath:    tree.FullCodePath,
		TemplateType:    tree.TemplateType,
		Version:         tree.Version,
		VersionNum:      tree.VersionNum,
		HubFullCodePath: tree.HubFullCodePath,
		HubVersionNum:   tree.HubVersionNum,
		RunCount:        tree.RunCount,
	}

	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && tree.FullCodePath != "" && s.permissionService != nil {
		permCtx, err := s.loadWorkspacePermissionContext(ctx, tree.FullCodePath)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 查询权限失败: fullCodePath=%s, error=%v，继续返回详情（无权限信息）", tree.FullCodePath, err)
			resp.Permissions = make(map[string]bool)
		} else {
			var nodeTypeStr string
			if tree.Type == model.ServiceTreeTypePackage {
				nodeTypeStr = "package"
			} else if tree.Type == model.ServiceTreeTypeFunction {
				nodeTypeStr = "function"
			}

			resp.Permissions = s.buildQueryNodePermissions(nodeTypeStr, tree.TemplateType, tree.FullCodePath, permCtx, username)
		}
	} else {
		resp.Permissions = make(map[string]bool)
	}

	return resp, nil
}

func getPackageInfoImpl(s *ServiceTreeService, ctx context.Context, req *dto.GetPackageInfoReq) (*dto.GetPackageInfoResp, error) {
	var tree *model.ServiceTree
	var err error

	if req.ID > 0 {
		tree, err = s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	} else {
		return nil, fmt.Errorf("必须提供 ID 或 full_code_path")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("目录不存在")
		}
		return nil, fmt.Errorf("获取目录失败: %w", err)
	}

	if tree.Type != model.ServiceTreeTypePackage {
		return nil, fmt.Errorf("该接口仅用于获取目录信息，函数信息请使用函数详情接口")
	}

	resp := &dto.GetPackageInfoResp{
		ID:           tree.ID,
		Name:         tree.Name,
		Code:         tree.Code,
		FullCodePath: tree.FullCodePath,
	}

	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && tree.FullCodePath != "" && s.permissionService != nil {
		permCtx, err := s.loadWorkspacePermissionContext(ctx, tree.FullCodePath)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 查询权限失败: fullCodePath=%s, error=%v，继续返回目录信息（无权限信息）", tree.FullCodePath, err)
			resp.Permissions = make(map[string]bool)
		} else {
			resp.Permissions = s.buildQueryNodePermissions("package", "", tree.FullCodePath, permCtx, username)
		}
	} else {
		resp.Permissions = make(map[string]bool)
	}

	return resp, nil
}
