package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (s *ServiceTreeService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	if req == nil {
		return s.mutationService.UpdateServiceTreeMetadata(ctx, req)
	}
	oldNode := s.getServiceTreeForAudit(req.ID)
	if err := s.mutationService.UpdateServiceTreeMetadata(ctx, req); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.updated", oldNode, s.getServiceTreeForAudit(req.ID))
	return nil
}

func (s *ServiceTreeService) UpdatePackage(ctx context.Context, req *dto.UpdatePackageReq) error {
	if req == nil {
		return s.mutationService.UpdatePackage(ctx, req)
	}
	oldNode := s.getServiceTreeForAudit(req.ID)
	if err := s.mutationService.UpdatePackage(ctx, req); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.updated", oldNode, s.getServiceTreeForAudit(req.ID))
	return nil
}

func (s *ServiceTreeService) UpdateFunction(ctx context.Context, req *dto.UpdateFunctionReq) error {
	if req == nil {
		return s.mutationService.UpdateFunction(ctx, req)
	}
	oldNode := s.getServiceTreeForAudit(req.ID)
	if err := s.mutationService.UpdateFunction(ctx, req); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.updated", oldNode, s.getServiceTreeForAudit(req.ID))
	return nil
}

func (s *ServiceTreeService) UpdateDocs(ctx context.Context, req *dto.UpdateDocsReq) error {
	if req == nil {
		return s.mutationService.UpdateDocs(ctx, req)
	}
	oldNode := s.getServiceTreeForAudit(req.ID)
	if err := s.mutationService.UpdateDocs(ctx, req); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.updated", oldNode, s.getServiceTreeForAudit(req.ID))
	return nil
}

func (s *ServiceTreeService) DeletePackage(ctx context.Context, id int64) error {
	oldNode := s.getServiceTreeForAudit(id)
	if err := s.mutationService.DeletePackage(ctx, id); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.deleted", oldNode, nil)
	return nil
}

func (s *ServiceTreeService) DeleteFunction(ctx context.Context, id int64) error {
	oldNode := s.getServiceTreeForAudit(id)
	if err := s.mutationService.DeleteFunction(ctx, id); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.deleted", oldNode, nil)
	return nil
}

func (s *ServiceTreeService) DeleteDocs(ctx context.Context, id int64) error {
	oldNode := s.getServiceTreeForAudit(id)
	if err := s.mutationService.DeleteDocs(ctx, id); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.deleted", oldNode, nil)
	return nil
}

func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	oldNode := s.getServiceTreeForAudit(id)
	if err := s.mutationService.DeleteServiceTree(ctx, id); err != nil {
		return err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.deleted", oldNode, nil)
	return nil
}
