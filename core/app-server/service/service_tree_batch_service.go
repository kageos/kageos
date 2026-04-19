package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type serviceTreeBatchService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	appService       *AppService
}

func newServiceTreeBatchService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
) *serviceTreeBatchService {
	return &serviceTreeBatchService{
		serviceTreeRepo:  serviceTreeRepo,
		runtimeWorkspace: runtimeWorkspace,
		appService:       appService,
	}
}

func (s *serviceTreeBatchService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return executeBatchCreateDirectoryTree(ctx, s.serviceTreeRepo, s.runtimeWorkspace, req)
}

func (s *serviceTreeBatchService) BatchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	return executeBatchWriteFiles(ctx, s.runtimeWorkspace, s.appService, req)
}
