package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type serviceTreeHubService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	functionRepo     *repository.FunctionRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	appService       *AppService
}

func newServiceTreeHubService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	functionRepo *repository.FunctionRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
) *serviceTreeHubService {
	return &serviceTreeHubService{
		serviceTreeRepo:  serviceTreeRepo,
		functionRepo:     functionRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		appService:       appService,
	}
}

func (h *serviceTreeHubService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return copyServiceTreeImpl(h, ctx, req)
}

func (h *serviceTreeHubService) copyFromLocal(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	return copyFromLocalImpl(h, ctx, req, targetApp)
}

func (h *serviceTreeHubService) copyFromHub(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	return copyFromHubImpl(h, ctx, req, targetApp)
}

func (h *serviceTreeHubService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	return publishDirectoryToHubImpl(h, ctx, req)
}

func (h *serviceTreeHubService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	return pushDirectoryToHubImpl(h, ctx, req)
}

func (h *serviceTreeHubService) GetHubPushFormInfo(ctx context.Context, req *dto.GetHubPushFormInfoReq) (*dto.GetHubPushFormInfoResp, error) {
	return getHubPushFormInfoImpl(h, ctx, req)
}

func (h *serviceTreeHubService) buildDirectoryTree(
	rootTree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
	refIDToFunction map[int64]*model.Function,
) *dto.DirectoryTreeNode {
	return buildDirectoryTreeImpl(h, rootTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
}

func (h *serviceTreeHubService) buildDirectoryTreeNode(
	tree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
	refIDToFunction map[int64]*model.Function,
) *dto.DirectoryTreeNode {
	return buildDirectoryTreeNodeImpl(h, tree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
}

func (h *serviceTreeHubService) installDirectoryTreeFromHubSnapshot(
	ctx context.Context,
	tree *dto.DirectoryTreeNode,
	targetApp *model.App,
	targetPath string,
	hubFullCodePath string,
	hubVersionNum int,
	hubDirectoryName string,
	successMessagePrefix string,
) (*dto.PullDirectoryFromHubResp, error) {
	return installDirectoryTreeFromHubSnapshotImpl(h, ctx, tree, targetApp, targetPath, hubFullCodePath, hubVersionNum, hubDirectoryName, successMessagePrefix)
}

func (h *serviceTreeHubService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	return pullDirectoryFromHubImpl(h, ctx, req)
}

func (h *serviceTreeHubService) ImportHubDirectoryBundle(ctx context.Context, req *dto.ImportHubDirectoryBundleReq) (*dto.PullDirectoryFromHubResp, error) {
	return importHubDirectoryBundleImpl(h, ctx, req)
}

func (h *serviceTreeHubService) countFilesInTree(node *dto.DirectoryTreeNode) int {
	return countFilesInTreeImpl(h, node)
}

func (h *serviceTreeHubService) logDirectoryTree(ctx context.Context, node *dto.DirectoryTreeNode, level int) {
	logDirectoryTreeImpl(h, ctx, node, level)
}

func (h *serviceTreeHubService) buildItemsFromTree(
	node *dto.DirectoryTreeNode,
	targetBasePath string,
	directoryItems *[]*dto.DirectoryScaffoldItem,
	fileItems *[]*dto.FileWriteItem,
) {
	buildItemsFromTreeImpl(h, node, targetBasePath, directoryItems, fileItems)
}

func (h *serviceTreeHubService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	return getHubInfoImpl(h, ctx, req)
}

func (h *serviceTreeHubService) batchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return executeBatchCreateDirectoryTree(ctx, h.serviceTreeRepo, h.runtimeWorkspace, req)
}

func (h *serviceTreeHubService) batchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	return executeBatchWriteFiles(ctx, h.runtimeWorkspace, h.appService, h.appRepo, req)
}

func (h *serviceTreeHubService) getDirectoryFilesFromRuntimeRecursively(
	ctx context.Context,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	return readDirectoryFilesFromRuntimeRecursively(ctx, h.serviceTreeRepo, h.runtimeWorkspace, appID, rootDirectoryPath)
}
