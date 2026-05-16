package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type serviceTreeQueryView struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
}

func newServiceTreeQueryView(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
) *serviceTreeQueryView {
	return &serviceTreeQueryView{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
	}
}

func (q *serviceTreeQueryView) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	var trees []*model.ServiceTree
	var err error
	if nodeType != "" {
		trees, err = q.serviceTreeRepo.BuildServiceTreeByType(appModel.ID, nodeType)
	} else {
		trees, err = q.serviceTreeRepo.BuildServiceTree(appModel.ID)
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

	logger.Debugf(ctx, "[ServiceTreeService] 按 MVP 模式返回全量可访问服务树: app_id=%d", appModel.ID)

	rootResp := q.convertToGetServiceTreeResp(rootNode)

	logger.Debugf(ctx, "[ServiceTreeService] 服务树转换完成: root_id=%d, children_count=%d",
		rootNode.ID, len(rootResp.Children))

	return []*dto.GetServiceTreeResp{rootResp}, nil
}

func (q *serviceTreeQueryView) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	user, appCode, err := resolveUserAppFromResourcePath(req.ResourcePath, req.User, req.App)
	if err != nil {
		return nil, err
	}
	req.User = user
	req.App = appCode

	appModel, err := q.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", req.User, req.App)
		}
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	appInfo := dto.AppInfo{
		ID:        appModel.ID,
		User:      appModel.User,
		Code:      appModel.Code,
		Name:      appModel.Name,
		Status:    appModel.Status,
		Version:   appModel.Version,
		NatsID:    appModel.NatsID,
		HostID:    appModel.HostID,
		IsPublic:  appModel.IsPublic,
		Admins:    appModel.Admins,
		Type:      int(appModel.Type),
		CreatedAt: time.Time(appModel.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt: time.Time(appModel.UpdatedAt).Format("2006-01-02 15:04:05"),
	}

	serviceTreeResp, err := q.getServiceTreeByAppModel(ctx, appModel, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录树失败: %w", err)
	}

	expandedKeys := q.calculateExpandedKeys(serviceTreeResp)

	return &dto.GetAppWithServiceTreeResp{
		App:          appInfo,
		ServiceTree:  serviceTreeResp,
		ExpandedKeys: expandedKeys,
	}, nil
}

func (q *serviceTreeQueryView) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	var tree *model.ServiceTree
	var err error

	if req.ID > 0 {
		tree, err = q.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = q.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
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
		ID:           tree.ID,
		Name:         tree.Name,
		Code:         tree.Code,
		Type:         tree.Type,
		Description:  tree.Description,
		Tags:         tree.Tags,
		AppID:        tree.AppID,
		RefID:        tree.RefID,
		FullCodePath: tree.FullCodePath,
		TemplateType: tree.TemplateType,
		Version:      tree.Version,
		VersionNum:   tree.VersionNum,
		RunCount:     tree.RunCount,
	}

	return resp, nil
}

func (q *serviceTreeQueryView) convertToGetServiceTreeResp(tree *model.ServiceTree) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:           tree.ID,
		Name:         tree.Name,
		Code:         tree.Code,
		RefID:        tree.RefID,
		Type:         tree.Type,
		Description:  tree.Description,
		Tags:         tree.Tags,
		Admins:       tree.Admins,
		Owner:        tree.CreatedBy,
		AppID:        tree.AppID,
		FullCodePath: tree.FullCodePath,
		TemplateType: tree.TemplateType,
		Version:      tree.Version,
		VersionNum:   tree.VersionNum,
		RunCount:     tree.RunCount,
	}

	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			childResp := q.convertToGetServiceTreeResp(child)
			resp.Children = append(resp.Children, childResp)
		}
	}

	if tree.Type == model.ServiceTreeTypePackage {
		resp.HasFunction = q.hasFunctionInDirectChildren(tree)
	}

	return resp
}

func (q *serviceTreeQueryView) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	expandedKeysMap := make(map[int64]bool)

	for _, tree := range trees {
		segments := strings.Split(strings.Trim(tree.FullCodePath, "/"), "/")
		if tree.Type == model.ServiceTreeTypePackage && len(segments) == 2 {
			expandedKeysMap[tree.ID] = true
		}
	}

	expandedKeys := make([]int64, 0, len(expandedKeysMap))
	for id := range expandedKeysMap {
		expandedKeys = append(expandedKeys, id)
	}

	return expandedKeys
}

func (q *serviceTreeQueryView) hasFunctionInDirectChildren(node *model.ServiceTree) bool {
	if node == nil {
		return false
	}
	for _, child := range node.Children {
		if child.Type == model.ServiceTreeTypeFunction {
			return true
		}
	}
	return false
}
