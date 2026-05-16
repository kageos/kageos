package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
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
		req.CurrentUser,
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

func (s *serviceTreeSearchService) SearchResources(
	_ context.Context,
	req *dto.SearchResourcesReq,
) (*dto.SearchResourcesResp, error) {
	page, pageSize := normalizeSearchResourcesPagination(req.Page, req.PageSize)
	fetchSize := calculateSearchResourcesFetchSize(page, pageSize, req.Keyword)
	nodeTypes := parseSearchResourceTypes(req.ResourceType)

	trees, total, err := s.serviceTreeRepo.SearchResources(
		req.CurrentUser,
		req.User,
		req.App,
		req.Keyword,
		nodeTypes,
		page,
		fetchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索资源失败: %w", err)
	}

	trees = rankAndLimitSearchResources(trees, req.Keyword, page, pageSize)

	return &dto.SearchResourcesResp{
		Items:    buildResourceSearchResults(trees, req.Keyword),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func parseSearchResourceTypes(resourceType string) []string {
	switch resourceType {
	case "", "all":
		return []string{
			model.ServiceTreeTypePackage,
			model.ServiceTreeTypeFunction,
			model.ServiceTreeTypeDocs,
		}
	case permission.ResourceTypeDirectory, model.ServiceTreeTypePackage:
		return []string{model.ServiceTreeTypePackage}
	case model.ServiceTreeTypeFunction:
		return []string{model.ServiceTreeTypeFunction}
	case model.ServiceTreeTypeDocs, "doc", "document":
		return []string{model.ServiceTreeTypeDocs}
	default:
		return []string{
			model.ServiceTreeTypePackage,
			model.ServiceTreeTypeFunction,
			model.ServiceTreeTypeDocs,
		}
	}
}
