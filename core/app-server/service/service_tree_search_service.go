package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type serviceTreeSearchService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
}

func newServiceTreeSearchService(serviceTreeRepo *repository.ServiceTreeRepository) *serviceTreeSearchService {
	return &serviceTreeSearchService{serviceTreeRepo: serviceTreeRepo}
}

func (s *serviceTreeSearchService) SearchFunctions(
	_ context.Context,
	req *dto.SearchFunctionsReq,
) (*dto.SearchFunctionsResp, error) {
	page, pageSize := normalizeSearchFunctionsPagination(req.Page, req.PageSize)
	fetchSize := calculateSearchFunctionsFetchSize(page, pageSize, req.Keyword)

	trees, total, err := s.serviceTreeRepo.SearchFunctions(
		req.User,
		req.App,
		req.Keyword,
		req.TemplateType,
		page,
		fetchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索函数失败: %w", err)
	}

	trees = rankAndLimitSearchFunctions(trees, req.Keyword, page, pageSize)

	return &dto.SearchFunctionsResp{
		Functions: buildFunctionSearchResults(trees),
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}
