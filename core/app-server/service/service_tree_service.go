package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
)

// extractVersionNum 从版本号字符串中提取数字部分（如 "v1" -> 1, "v20" -> 20）
func extractVersionNumForServiceTree(version string) int {
	if version == "" {
		return 0
	}
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	num, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return num
}

type ServiceTreeService struct {
	queryView          *serviceTreeQueryView
	workspaceService   *serviceTreeWorkspaceService
	searchService      *serviceTreeSearchService
	hubService         *serviceTreeHubService
	mutationService    *serviceTreeMutationService
	specialNodeService *serviceTreeSpecialNodeService
	functionService    *serviceTreeFunctionService
	packageService     *serviceTreePackageService
	batchService       *serviceTreeBatchService
	directoryBundle    *serviceTreeDirectoryBundleService
}

func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	functionRepo *repository.FunctionRepository,
	appRepo *repository.AppRepository,
	appCall *appcall.Client,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
	permissionService *PermissionService,
	docService *DocService,
	boardPostRepo *repository.BoardPostRepository,
) *ServiceTreeService {
	runtimeWorkspace := newRuntimeWorkspaceBridge(appRepo, appCall)
	queryView := newServiceTreeQueryView(serviceTreeRepo, appRepo, permissionService)

	return &ServiceTreeService{
		queryView:          queryView,
		workspaceService:   newServiceTreeWorkspaceService(serviceTreeRepo, fileSnapshotRepo, runtimeWorkspace, queryView),
		searchService:      newServiceTreeSearchService(serviceTreeRepo),
		hubService:         newServiceTreeHubService(serviceTreeRepo, functionRepo, appRepo, runtimeWorkspace, appService),
		mutationService:    newServiceTreeMutationService(serviceTreeRepo, appRepo, runtimeWorkspace, docService, boardPostRepo),
		specialNodeService: newServiceTreeSpecialNodeService(serviceTreeRepo, appRepo, docService),
		functionService:    newServiceTreeFunctionService(serviceTreeRepo, appRepo, appService),
		packageService:     newServiceTreePackageService(serviceTreeRepo, appRepo, runtimeWorkspace),
		batchService:       newServiceTreeBatchService(serviceTreeRepo, runtimeWorkspace, appService),
		directoryBundle:    newServiceTreeDirectoryBundleService(serviceTreeRepo, appRepo, runtimeWorkspace, appService),
	}
}

func (s *ServiceTreeService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	return s.packageService.CreatePackage(ctx, req)
}

func (s *ServiceTreeService) CreateFunction(ctx context.Context, req *dto.CreateFunctionReq) (*dto.CreateFunctionResp, error) {
	return s.functionService.CreateFunction(ctx, req)
}

func (s *ServiceTreeService) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	return s.queryView.GetAppWithServiceTree(ctx, req)
}

func (s *ServiceTreeService) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	return s.queryView.GetServiceTreeDetail(ctx, req)
}

func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return s.hubService.CopyServiceTree(ctx, req)
}

func (s *ServiceTreeService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	return s.hubService.PublishDirectoryToHub(ctx, req)
}

func (s *ServiceTreeService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	return s.hubService.PushDirectoryToHub(ctx, req)
}

func (s *ServiceTreeService) GetHubPushFormInfo(ctx context.Context, req *dto.GetHubPushFormInfoReq) (*dto.GetHubPushFormInfoResp, error) {
	return s.hubService.GetHubPushFormInfo(ctx, req)
}

func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return s.batchService.BatchCreateDirectoryTree(ctx, req)
}

func (s *ServiceTreeService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	return s.functionService.AddFunctions(ctx, req)
}

func (s *ServiceTreeService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	return s.functionService.ProcessFunctionGenResult(ctx, req)
}

func (s *ServiceTreeService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	return s.hubService.PullDirectoryFromHub(ctx, req)
}

func (s *ServiceTreeService) ImportHubDirectoryBundle(ctx context.Context, req *dto.ImportHubDirectoryBundleReq) (*dto.PullDirectoryFromHubResp, error) {
	return s.hubService.ImportHubDirectoryBundle(ctx, req)
}

func (s *ServiceTreeService) ExportDirectoryBundle(ctx context.Context, req *dto.ExportDirectoryBundleReq) (*dto.DirectoryBundle, error) {
	return s.directoryBundle.ExportDirectoryBundle(ctx, req)
}

func (s *ServiceTreeService) ImportDirectoryBundle(ctx context.Context, req *dto.ImportDirectoryBundleReq) (*dto.ImportDirectoryBundleResp, error) {
	return s.directoryBundle.ImportDirectoryBundle(ctx, req)
}

func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	return s.batchService.BatchWriteFiles(ctx, req)
}

func (s *ServiceTreeService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	return s.hubService.GetHubInfo(ctx, req)
}

func (s *ServiceTreeService) SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	return s.searchService.SearchFunctions(ctx, req)
}

func (s *ServiceTreeService) SearchResources(ctx context.Context, req *dto.SearchResourcesReq) (*dto.SearchResourcesResp, error) {
	return s.searchService.SearchResources(ctx, req)
}
