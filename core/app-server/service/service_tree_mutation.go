package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (s *ServiceTreeService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	return s.mutationService.UpdateServiceTreeMetadata(ctx, req)
}

func (s *ServiceTreeService) UpdatePackage(ctx context.Context, req *dto.UpdatePackageReq) error {
	return s.mutationService.UpdatePackage(ctx, req)
}

func (s *ServiceTreeService) UpdateFunction(ctx context.Context, req *dto.UpdateFunctionReq) error {
	return s.mutationService.UpdateFunction(ctx, req)
}

func (s *ServiceTreeService) UpdateDocs(ctx context.Context, req *dto.UpdateDocsReq) error {
	return s.mutationService.UpdateDocs(ctx, req)
}

func (s *ServiceTreeService) DeletePackage(ctx context.Context, id int64) error {
	return s.mutationService.DeletePackage(ctx, id)
}

func (s *ServiceTreeService) DeleteFunction(ctx context.Context, id int64) error {
	return s.mutationService.DeleteFunction(ctx, id)
}

func (s *ServiceTreeService) DeleteDocs(ctx context.Context, id int64) error {
	return s.mutationService.DeleteDocs(ctx, id)
}

func (s *ServiceTreeService) UpdateBoard(ctx context.Context, req *dto.UpdateBoardReq) error {
	return s.mutationService.UpdateBoard(ctx, req)
}

func (s *ServiceTreeService) DeleteBoard(ctx context.Context, id int64) error {
	return s.mutationService.DeleteBoard(ctx, id)
}

func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	return s.mutationService.DeleteServiceTree(ctx, id)
}
