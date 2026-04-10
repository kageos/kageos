package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type serviceTreeQueryView struct {
	serviceTreeRepo   *repository.ServiceTreeRepository
	appRepo           *repository.AppRepository
	permissionService *PermissionService
}

func newServiceTreeQueryView(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	permissionService *PermissionService,
) *serviceTreeQueryView {
	return &serviceTreeQueryView{
		serviceTreeRepo:   serviceTreeRepo,
		appRepo:           appRepo,
		permissionService: permissionService,
	}
}

func (q *serviceTreeQueryView) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	return getServiceTreeByAppModelImpl(q, ctx, appModel, nodeType)
}

func (q *serviceTreeQueryView) filterTreeByPermission(rootResp *dto.GetServiceTreeResp) *dto.GetServiceTreeResp {
	rootResp.Children = q.collectVisibleChildren(rootResp.Children)
	return rootResp
}

func (q *serviceTreeQueryView) collectVisibleChildren(children []*dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	out := make([]*dto.GetServiceTreeResp, 0, len(children))
	for _, child := range children {
		out = append(out, q.collectVisibleFromNode(child)...)
	}
	return out
}

func (q *serviceTreeQueryView) collectVisibleFromNode(node *dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	hasAnyTrue := false
	if node.Permissions != nil {
		for _, v := range node.Permissions {
			if v {
				hasAnyTrue = true
				break
			}
		}
	}
	if !hasAnyTrue {
		return q.collectVisibleChildren(node.Children)
	}
	node.Children = q.collectVisibleChildren(node.Children)
	return []*dto.GetServiceTreeResp{node}
}

func (q *serviceTreeQueryView) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	return getAppWithServiceTreeImpl(q, ctx, req)
}

func (q *serviceTreeQueryView) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	return getServiceTreeDetailImpl(q, ctx, req)
}

func (q *serviceTreeQueryView) convertToGetServiceTreeResp(ctx context.Context, tree *model.ServiceTree, permissionsMap map[string]map[string]bool, isAdmin bool) *dto.GetServiceTreeResp {
	return convertToGetServiceTreeRespImpl(q, ctx, tree, permissionsMap, isAdmin)
}

func (q *serviceTreeQueryView) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	return calculateExpandedKeysImpl(trees)
}

func (q *serviceTreeQueryView) calculatePermissions(ctx context.Context, user, app string, trees []*model.ServiceTree, admins string, username string) (map[string]map[string]bool, error) {
	return calculatePermissionsImpl(q, ctx, user, app, trees, admins, username)
}

func (q *serviceTreeQueryView) applyPermissionInheritance(
	nodeType string,
	templateType string,
	parentPerms map[string]bool,
	nodePerms map[string]bool,
) {
	applyPermissionInheritanceImpl(nodeType, templateType, parentPerms, nodePerms)
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

func (q *serviceTreeQueryView) getPermissionActionsForNode(nodeType string, templateType string) []string {
	return getPermissionActionsForNodeImpl(nodeType, templateType)
}

func (q *serviceTreeQueryView) GetServiceTreeByFullPath(fullPath string) (*model.ServiceTree, error) {
	return q.serviceTreeRepo.GetServiceTreeByFullPath(fullPath)
}
