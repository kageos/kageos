package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

type serviceTreeQueryView struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	teamAccess      *TeamAccessService
}

const (
	serviceTreeScheduledAgentBadgeTimeout     = 1500 * time.Millisecond
	serviceTreeScheduledAgentBadgeConcurrency = 10
)

func newServiceTreeQueryView(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	teamAccess *TeamAccessService,
) *serviceTreeQueryView {
	return &serviceTreeQueryView{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		teamAccess:      teamAccess,
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

	logger.Debugf(ctx, "[ServiceTreeService] 加载服务树: app_id=%d, hide_unauthorized_nodes=%t", appModel.ID, appModel.HideUnauthorizedNodes)

	permissionsByPath, err := q.permissionsByPath(ctx, appModel, trees)
	if err != nil {
		return nil, err
	}

	rootResp := q.convertToGetServiceTreeResp(rootNode, permissionsByPath)
	if appModel.HideUnauthorizedNodes {
		rootResp = filterVisibleServiceTree(rootResp)
	}
	if rootResp == nil {
		return []*dto.GetServiceTreeResp{}, nil
	}
	q.annotateScheduledAgentTaskCounts(ctx, rootResp)

	logger.Debugf(ctx, "[ServiceTreeService] 服务树转换完成: root_id=%d, children_count=%d",
		rootNode.ID, len(rootResp.Children))

	return []*dto.GetServiceTreeResp{rootResp}, nil
}

func (q *serviceTreeQueryView) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	appModel, err := q.appRepo.GetAppByUserName(user, appCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", user, appCode)
		}
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	appInfo := dto.AppInfo{
		ID:                    appModel.ID,
		User:                  appModel.User,
		Code:                  appModel.Code,
		Name:                  appModel.Name,
		Status:                appModel.Status,
		Version:               appModel.Version,
		NatsID:                appModel.NatsID,
		HostID:                appModel.HostID,
		IsPublic:              appModel.IsPublic,
		HideUnauthorizedNodes: appModel.HideUnauthorizedNodes,
		Admins:                appModel.Admins,
		Type:                  int(appModel.Type),
		CreatedAt:             time.Time(appModel.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt:             time.Time(appModel.UpdatedAt).Format("2006-01-02 15:04:05"),
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
			if message := q.missingAppRootMessage(req.FullCodePath); message != "" {
				return nil, fmt.Errorf("%s", message)
			}
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("获取服务目录失败: %w", err)
	}

	resp := &dto.GetServiceTreeDetailResp{
		ID:                 tree.ID,
		Name:               tree.Name,
		Code:               tree.Code,
		Type:               tree.Type,
		Description:        tree.Description,
		Tags:               tree.Tags,
		Connectors:         splitConnectorCodes(tree.Connectors),
		ConnectorEndpoints: splitConnectorEndpoints(tree.ConnectorEndpoints),
		AppID:              tree.AppID,
		RefID:              tree.RefID,
		FullCodePath:       tree.FullCodePath,
		TemplateType:       tree.TemplateType,
		Version:            tree.Version,
		VersionNum:         tree.VersionNum,
		RunCount:           tree.RunCount,
	}

	return resp, nil
}

func (q *serviceTreeQueryView) missingAppRootMessage(fullCodePath string) string {
	user, appCode, rootPath, ok := parseAppRootFullCodePath(fullCodePath)
	if !ok || q.appRepo == nil {
		return ""
	}
	appModel, err := q.appRepo.GetAppByUserName(user, appCode)
	if err != nil || appModel == nil {
		return ""
	}
	return fmt.Sprintf("工作空间根节点缺失: %s（应用存在但 service_tree 根节点不存在，请执行应用根节点初始化修复）", rootPath)
}

func parseAppRootFullCodePath(fullCodePath string) (user string, app string, rootPath string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", "", false
	}
	user = strings.TrimSpace(parts[0])
	app = strings.TrimSpace(parts[1])
	return user, app, "/" + user + "/" + app, true
}

func (q *serviceTreeQueryView) permissionsByPath(ctx context.Context, appModel *model.App, trees []*model.ServiceTree) (map[string]*access.Result, error) {
	paths := collectServiceTreePaths(trees)
	if q.teamAccess == nil {
		results := make(map[string]*access.Result, len(paths))
		for _, path := range paths {
			normalized := access.NormalizeResourcePath(path)
			results[normalized] = &access.Result{ResourcePath: normalized, Permissions: access.RolePermissions(access.RoleOwner)}
		}
		return results, nil
	}
	return q.teamAccess.PermissionsForTree(ctx, appModel.User, appModel.Code, contextx.GetRequestUser(ctx), paths)
}

func collectServiceTreePaths(trees []*model.ServiceTree) []string {
	paths := make([]string, 0)
	var walk func(nodes []*model.ServiceTree)
	walk = func(nodes []*model.ServiceTree) {
		for _, node := range nodes {
			if node == nil {
				continue
			}
			paths = append(paths, node.FullCodePath)
			walk(node.Children)
		}
	}
	walk(trees)
	return paths
}

func (q *serviceTreeQueryView) convertToGetServiceTreeResp(tree *model.ServiceTree, permissionsByPath map[string]*access.Result) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:                 tree.ID,
		Name:               tree.Name,
		Code:               tree.Code,
		RefID:              tree.RefID,
		Type:               tree.Type,
		Description:        tree.Description,
		Tags:               tree.Tags,
		Connectors:         splitConnectorCodes(tree.Connectors),
		ConnectorEndpoints: splitConnectorEndpoints(tree.ConnectorEndpoints),
		Admins:             tree.Admins,
		Owner:              tree.CreatedBy,
		AppID:              tree.AppID,
		FullCodePath:       tree.FullCodePath,
		TemplateType:       tree.TemplateType,
		Version:            tree.Version,
		VersionNum:         tree.VersionNum,
		RunCount:           tree.RunCount,
	}
	if permissionResult := permissionsByPath[access.NormalizeResourcePath(tree.FullCodePath)]; permissionResult != nil {
		resp.Permissions = permissionResult.Permissions
		resp.RoleCodes = permissionResult.RoleCodes
		resp.InheritedFrom = permissionResult.InheritedFrom
		resp.ExpiresAt = permissionResult.ExpiresAt
	} else {
		resp.Permissions = access.EmptyPermissionSet()
	}

	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			childResp := q.convertToGetServiceTreeResp(child, permissionsByPath)
			resp.Children = append(resp.Children, childResp)
		}
	}

	if tree.Type == model.ServiceTreeTypePackage {
		resp.HasFunction = q.hasFunctionInDirectChildren(tree)
	}

	return resp
}

func (q *serviceTreeQueryView) annotateScheduledAgentTaskCounts(ctx context.Context, root *dto.GetServiceTreeResp) {
	if root == nil {
		return
	}
	packages := collectPackageServiceTreeResp(root)
	if len(packages) == 0 {
		return
	}
	client := newServiceTreeScheduleClient()
	if client == nil {
		return
	}

	badgeCtx, cancel := context.WithTimeout(ctx, serviceTreeScheduledAgentBadgeTimeout)
	defer cancel()
	counts, failed := loadScheduledAgentTaskCountsByDirectory(badgeCtx, client, packages)
	if failed > 0 {
		logger.Debugf(ctx, "[ServiceTreeService] Agent 任务徽章部分加载失败: failed=%d total=%d", failed, len(packages))
	}
	applyScheduledAgentTaskSubtreeCounts(root, counts)
}

func collectPackageServiceTreeResp(root *dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	out := make([]*dto.GetServiceTreeResp, 0)
	var walk func(*dto.GetServiceTreeResp)
	walk = func(node *dto.GetServiceTreeResp) {
		if node == nil {
			return
		}
		if node.Type == model.ServiceTreeTypePackage {
			out = append(out, node)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)
	return out
}

func loadScheduledAgentTaskCountsByDirectory(ctx context.Context, client serviceTreeScheduleClient, packages []*dto.GetServiceTreeResp) (map[string]int, int) {
	counts := make(map[string]int, len(packages))
	if client == nil || len(packages) == 0 {
		return counts, 0
	}

	sem := make(chan struct{}, serviceTreeScheduledAgentBadgeConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := 0

	for _, node := range packages {
		path := access.NormalizeResourcePath(node.FullCodePath)
		if path == "" {
			continue
		}
		wg.Add(1)
		go func(resourcePath string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			resp, err := client.ListTasks(ctx, scheduledsdk.ListTasksRequest{
				ExecutorKey:   ScheduledAgentSessionExecutorKey,
				ResourceScope: "workspace_directory",
				ResourceKey:   resourcePath,
				Page:          1,
				PageSize:      1,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				return
			}
			if resp != nil {
				counts[resourcePath] = int(resp.Total)
			}
		}(path)
	}
	wg.Wait()
	return counts, failed
}

func applyScheduledAgentTaskSubtreeCounts(node *dto.GetServiceTreeResp, exactCounts map[string]int) int {
	if node == nil {
		return 0
	}
	total := 0
	for _, child := range node.Children {
		total += applyScheduledAgentTaskSubtreeCounts(child, exactCounts)
	}
	if node.Type == model.ServiceTreeTypePackage {
		total += exactCounts[access.NormalizeResourcePath(node.FullCodePath)]
		node.ScheduledAgentTasks = total
	}
	return total
}

func filterVisibleServiceTree(node *dto.GetServiceTreeResp) *dto.GetServiceTreeResp {
	if node == nil {
		return nil
	}
	filteredChildren := make([]*dto.GetServiceTreeResp, 0, len(node.Children))
	for _, child := range node.Children {
		if filteredChild := filterVisibleServiceTree(child); filteredChild != nil {
			filteredChildren = append(filteredChildren, filteredChild)
		}
	}
	node.Children = filteredChildren
	if access.HasPermission(node.Permissions, access.ActionRead) || len(node.Children) > 0 {
		return node
	}
	return nil
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
