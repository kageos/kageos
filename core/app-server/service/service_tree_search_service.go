package service

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
)

type serviceTreeSearchService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	permission      *PermissionService
}

func newServiceTreeSearchService(serviceTreeRepo *repository.ServiceTreeRepository, permission *PermissionService) *serviceTreeSearchService {
	return &serviceTreeSearchService{serviceTreeRepo: serviceTreeRepo, permission: permission}
}

func (s *serviceTreeSearchService) SearchFunctions(
	ctx context.Context,
	req *dto.SearchFunctionsReq,
) (*dto.SearchFunctionsResp, error) {
	page, pageSize := normalizeSearchFunctionsPagination(req.Page, req.PageSize)
	fetchSize := calculateSearchFunctionsFetchSize(page, pageSize, req.Keyword, req.FullCodePath)

	trees, _, err := s.serviceTreeRepo.SearchFunctions(
		req.CurrentUser,
		req.User,
		req.App,
		req.Keyword,
		req.FullCodePath,
		req.TemplateType,
		page,
		fetchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索函数失败: %w", err)
	}

	trees = s.filterReadableTrees(ctx, req.CurrentUser, trees)
	trees = rankAndLimitSearchFunctions(trees, req.Keyword, page, pageSize)

	return &dto.SearchFunctionsResp{
		Functions: buildFunctionSearchResults(trees),
		Total:     int64(len(trees)),
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

func (s *serviceTreeSearchService) SearchResources(
	ctx context.Context,
	req *dto.SearchResourcesReq,
) (*dto.SearchResourcesResp, error) {
	page, pageSize := normalizeSearchResourcesPagination(req.Page, req.PageSize)
	fetchSize := calculateSearchResourcesFetchSize(page, pageSize, req.Keyword, req.FullCodePath)
	nodeTypes := parseSearchResourceTypes(req.ResourceType)

	trees, _, err := s.serviceTreeRepo.SearchResources(
		req.CurrentUser,
		req.User,
		req.App,
		req.Keyword,
		req.FullCodePath,
		nodeTypes,
		page,
		fetchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索资源失败: %w", err)
	}

	trees = s.filterReadableTrees(ctx, req.CurrentUser, trees)
	trees = rankAndLimitSearchResources(trees, req.Keyword, page, pageSize)

	return &dto.SearchResourcesResp{
		Items:    buildResourceSearchResults(trees, req.Keyword),
		Total:    int64(len(trees)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *serviceTreeSearchService) filterReadableTrees(ctx context.Context, username string, trees []*model.ServiceTree) []*model.ServiceTree {
	if s.permission == nil || username == "" {
		return trees
	}
	filtered := make([]*model.ServiceTree, 0, len(trees))
	for _, tree := range trees {
		if tree == nil {
			continue
		}
		ok, err := s.permission.HasPermission(ctx, username, tree.FullCodePath, access.ActionRead)
		if err == nil && ok {
			filtered = append(filtered, tree)
		}
	}
	return filtered
}

func parseSearchResourceTypes(resourceType string) []string {
	switch resourceType {
	case "", "all":
		return []string{
			model.ServiceTreeTypePackage,
			model.ServiceTreeTypeFunction,
			model.ServiceTreeTypeDocs,
		}
	case "directory", model.ServiceTreeTypePackage:
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
