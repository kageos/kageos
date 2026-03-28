package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func splitSearchKeywordsForRelevance(keyword string) []string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}

	parts := strings.Split(keyword, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func searchFunctionsRelevanceScore(tree *model.ServiceTree, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}

	score := 0
	nameLower := strings.ToLower(strings.TrimSpace(tree.Name))
	codeLower := strings.ToLower(strings.TrimSpace(tree.Code))
	descLower := strings.ToLower(tree.Description)
	tagsLower := strings.ToLower(tree.Tags)
	tagSlice := tree.GetTagsSlice()
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}

		if nameLower == k {
			score += 10
		} else if strings.Contains(nameLower, k) {
			score += 3
		}

		if codeLower == k {
			score += 6
		} else if strings.Contains(codeLower, k) {
			score += 2
		}

		if strings.Contains(descLower, k) {
			score += 1
		}

		tagExact := false
		for _, t := range tagSlice {
			if strings.ToLower(strings.TrimSpace(t)) == k {
				tagExact = true
				break
			}
		}
		if tagExact {
			score += 5
		} else if strings.Contains(tagsLower, k) {
			score += 2
		}
	}

	return score
}

func searchFunctionsImpl(
	s *ServiceTreeService,
	ctx context.Context,
	req *dto.SearchFunctionsReq,
) (*dto.SearchFunctionsResp, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	fetchSize := pageSize
	if req.Keyword != "" && req.Page == 1 {
		fetchSize = 200
		if fetchSize > pageSize*10 {
			fetchSize = pageSize * 10
		}
	}

	trees, total, err := s.serviceTreeRepo.SearchFunctions(
		req.User,
		req.App,
		req.Keyword,
		req.TemplateType,
		req.Page,
		fetchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索函数失败: %w", err)
	}

	if req.Keyword != "" && req.Page == 1 && len(trees) > 0 {
		keywords := splitSearchKeywordsForRelevance(req.Keyword)
		type scored struct {
			tree  *model.ServiceTree
			score int
		}

		scoredList := make([]scored, 0, len(trees))
		for _, t := range trees {
			scoredList = append(scoredList, scored{tree: t, score: searchFunctionsRelevanceScore(t, keywords)})
		}
		sort.Slice(scoredList, func(i, j int) bool {
			if scoredList[i].score != scoredList[j].score {
				return scoredList[i].score > scoredList[j].score
			}
			return scoredList[i].tree.RunCount > scoredList[j].tree.RunCount
		})

		trees = make([]*model.ServiceTree, 0, pageSize)
		for i := 0; i < len(scoredList) && i < pageSize; i++ {
			trees = append(trees, scoredList[i].tree)
		}
	} else if req.Page > 1 && req.Keyword != "" {
		if len(trees) > pageSize {
			trees = trees[:pageSize]
		}
	} else if len(trees) > pageSize {
		trees = trees[:pageSize]
	}

	functionResults := make([]*dto.FunctionSearchResult, 0, len(trees))
	for _, tree := range trees {
		result := &dto.FunctionSearchResult{
			Name:         tree.Name,
			Code:         tree.Code,
			Description:  tree.Description,
			TemplateType: tree.TemplateType,
			FullCodePath: tree.FullCodePath,
			RunCount:     tree.RunCount,
		}
		if tree.App != nil {
			result.AppID = tree.AppID
			result.AppUser = tree.App.User
			result.AppCode = tree.App.Code
		}
		if tree.Function != nil {
			result.ID = tree.Function.ID
			result.FullCodePath = tree.Function.Router
			if len(tree.Function.Request) > 0 {
				var reqArr []interface{}
				if err := json.Unmarshal(tree.Function.Request, &reqArr); err == nil {
					result.Request = reqArr
				}
			}
			if len(tree.Function.Response) > 0 {
				var respArr []interface{}
				if err := json.Unmarshal(tree.Function.Response, &respArr); err == nil {
					result.Response = respArr
				}
			}
		}
		functionResults = append(functionResults, result)
	}

	return &dto.SearchFunctionsResp{
		Functions: functionResults,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}
