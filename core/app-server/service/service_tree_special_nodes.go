package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (s *ServiceTreeService) CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	resp, err := s.specialNodeService.CreateDocs(ctx, req)
	if err != nil {
		return nil, err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.created", nil, s.getServiceTreeForAuditByPath(resp.FullCodePath))
	return resp, nil
}

func (s *ServiceTreeService) CreateDocsNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	resp, err := s.specialNodeService.CreateDocsNode(ctx, req)
	if err != nil {
		return nil, err
	}
	s.writeServiceTreeOperateLog(ctx, "service_tree.node.created", nil, s.getServiceTreeForAuditByPath(resp.FullCodePath))
	return resp, nil
}
