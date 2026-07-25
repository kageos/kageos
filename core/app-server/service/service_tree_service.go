package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/appcall"
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
	copyService        *serviceTreeCopyService
	mutationService    *serviceTreeMutationService
	specialNodeService *serviceTreeSpecialNodeService
	functionService    *serviceTreeFunctionService
	packageService     *serviceTreePackageService
	batchService       *serviceTreeBatchService
	capabilityBundle   *serviceTreeCapabilityBundleService
	teamAccessService  *TeamAccessService
}

func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	appCall *appcall.Client,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
	docService *DocService,
	teamAccessService *TeamAccessService,
) *ServiceTreeService {
	runtimeWorkspace := newRuntimeWorkspaceBridge(appRepo, appCall)
	queryView := newServiceTreeQueryView(serviceTreeRepo, appRepo, teamAccessService)
	capabilityBundle := newServiceTreeCapabilityBundleService(serviceTreeRepo, appRepo, runtimeWorkspace, appService, docService)

	return &ServiceTreeService{
		queryView:          queryView,
		workspaceService:   newServiceTreeWorkspaceService(serviceTreeRepo, fileSnapshotRepo, runtimeWorkspace, queryView),
		searchService:      newServiceTreeSearchService(serviceTreeRepo, teamAccessService),
		copyService:        newServiceTreeCopyService(serviceTreeRepo, appRepo, runtimeWorkspace, appService, capabilityBundle),
		mutationService:    newServiceTreeMutationService(serviceTreeRepo, appRepo, runtimeWorkspace, docService),
		specialNodeService: newServiceTreeSpecialNodeService(serviceTreeRepo, appRepo, docService),
		functionService:    newServiceTreeFunctionService(serviceTreeRepo, appRepo, appService),
		packageService:     newServiceTreePackageService(serviceTreeRepo, appRepo, runtimeWorkspace),
		batchService:       newServiceTreeBatchService(serviceTreeRepo, runtimeWorkspace, appService),
		capabilityBundle:   capabilityBundle,
		teamAccessService:  teamAccessService,
	}
}

func (s *ServiceTreeService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	resp, err := s.packageService.CreatePackage(ctx, req)
	if err != nil {
		return nil, err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.created", nil, s.getServiceTreeForAuditByPath(resp.FullCodePath))
	return resp, nil
}

func (s *ServiceTreeService) CreateFunction(ctx context.Context, req *dto.CreateFunctionReq) (*dto.CreateFunctionResp, error) {
	resp, err := s.functionService.CreateFunction(ctx, req)
	if err != nil {
		return nil, err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.created", nil, s.getServiceTreeForAuditByPath(resp.FullCodePath))
	return resp, nil
}

func (s *ServiceTreeService) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	return s.queryView.GetAppWithServiceTree(ctx, req)
}

func (s *ServiceTreeService) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	return s.queryView.GetServiceTreeDetail(ctx, req)
}

func (s *ServiceTreeService) BatchGetServiceTreeDetails(ctx context.Context, req *dto.BatchGetServiceTreeDetailsReq) (*dto.BatchGetServiceTreeDetailsResp, error) {
	return s.queryView.BatchGetServiceTreeDetails(ctx, req)
}

func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return s.copyService.CopyServiceTree(ctx, req)
}

func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return s.batchService.BatchCreateDirectoryTree(ctx, req)
}

func (s *ServiceTreeService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	var oldNode *model.ServiceTree
	var expectedPath string
	if req != nil {
		expectedPath = s.resolveAddFunctionsAuditPath(ctx, req)
		oldNode = s.getServiceTreeForAuditByPath(expectedPath)
	}
	resp, err := s.functionService.AddFunctions(ctx, req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.Success && expectedPath != "" {
		action := "service_tree.node.created"
		if oldNode != nil {
			action = "service_tree.node.updated"
		}
		s.writeServiceTreeOperateLog(ctx, action, oldNode, s.getServiceTreeForAuditByPath(expectedPath))
	}
	return resp, nil
}

func (s *ServiceTreeService) ExportCapabilityBundle(ctx context.Context, req *dto.ExportCapabilityBundleReq) (*dto.CapabilityBundle, error) {
	return s.capabilityBundle.ExportCapabilityBundle(ctx, req)
}

func (s *ServiceTreeService) InstallCapabilityBundle(ctx context.Context, req *dto.InstallCapabilityBundleReq) (*dto.InstallCapabilityBundleResp, error) {
	if req == nil {
		return nil, fmt.Errorf("导入目录请求不能为空")
	}
	opts := req.InstallCapabilityOptions
	return s.capabilityBundle.InstallCapabilityBundle(ctx, &opts, req.Bundle)
}

func (s *ServiceTreeService) InstallCapabilityBundleFromFile(ctx context.Context, opts *dto.InstallCapabilityOptions, filePath string) (*dto.InstallCapabilityBundleResp, error) {
	return s.capabilityBundle.InstallCapabilityBundleFromFile(ctx, opts, filePath)
}

func (s *ServiceTreeService) InstallCapabilityBundleFromURL(ctx context.Context, req *dto.InstallCapabilityBundleFromURLReq) (*dto.InstallCapabilityBundleResp, error) {
	if req == nil {
		return nil, fmt.Errorf("通过 URL 导入目录请求不能为空")
	}
	opts := req.InstallCapabilityOptions
	return s.capabilityBundle.InstallCapabilityBundleFromURL(ctx, &opts, req.BundleURL, req.InstallKey)
}

func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	return s.batchService.BatchWriteFiles(ctx, req)
}

func (s *ServiceTreeService) SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	return s.searchService.SearchFunctions(ctx, req)
}

func (s *ServiceTreeService) SearchResources(ctx context.Context, req *dto.SearchResourcesReq) (*dto.SearchResourcesResp, error) {
	return s.searchService.SearchResources(ctx, req)
}
