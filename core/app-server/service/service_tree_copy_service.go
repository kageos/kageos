package service

import (
	"context"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
)

type serviceTreeCopyService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	appService       *AppService
	capabilityBundle *serviceTreeCapabilityBundleService
}

func newServiceTreeCopyService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
	capabilityBundle *serviceTreeCapabilityBundleService,
) *serviceTreeCopyService {
	return &serviceTreeCopyService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		appService:       appService,
		capabilityBundle: capabilityBundle,
	}
}

func (h *serviceTreeCopyService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return copyServiceTreeImpl(h, ctx, req)
}

func (h *serviceTreeCopyService) copyFromLocal(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	return copyFromLocalImpl(h, ctx, req, targetApp)
}

func (h *serviceTreeCopyService) batchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return executeBatchCreateDirectoryTree(ctx, h.serviceTreeRepo, h.runtimeWorkspace, req)
}

func (h *serviceTreeCopyService) batchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	return executeBatchWriteFiles(ctx, h.runtimeWorkspace, h.appService, req)
}

func (h *serviceTreeCopyService) replaceDirectoryTree(
	ctx context.Context,
	req *dto.ReplaceDirectoryTreeReq,
) (*model.App, *dto.ReplaceDirectoryTreeResp, error) {
	return executeReplaceDirectoryTree(ctx, h.runtimeWorkspace, h.appService, req)
}

func (h *serviceTreeCopyService) getDirectoryFilesFromRuntimeRecursively(
	ctx context.Context,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	return readDirectoryFilesFromRuntimeRecursively(ctx, h.serviceTreeRepo, h.runtimeWorkspace, appID, rootDirectoryPath)
}
