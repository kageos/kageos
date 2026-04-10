package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (s *ServiceTreeService) CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	return s.specialNodeService.CreateDocs(ctx, req)
}

func (s *ServiceTreeService) CreateDocsNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	return s.specialNodeService.CreateDocsNode(ctx, req)
}

func (s *ServiceTreeService) CreateBoard(ctx context.Context, req *dto.CreateBoardReq) (*dto.CreateBoardResp, error) {
	return s.specialNodeService.CreateBoard(ctx, req)
}

func (s *ServiceTreeService) CreateBoardNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	return s.specialNodeService.CreateBoardNode(ctx, req)
}
