package service

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func normalizeSearchFunctionsPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}

func calculateSearchFunctionsFetchSize(page, pageSize int, keyword string) int {
	if keyword == "" || page != 1 {
		return pageSize
	}

	fetchSize := 200
	if fetchSize > pageSize*10 {
		fetchSize = pageSize * 10
	}
	return fetchSize
}

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

func rankAndLimitSearchFunctions(
	trees []*model.ServiceTree,
	keyword string,
	page int,
	pageSize int,
) []*model.ServiceTree {
	if pageSize <= 0 || len(trees) == 0 {
		return trees
	}

	if keyword != "" && page == 1 {
		keywords := splitSearchKeywordsForRelevance(keyword)
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

		limitedTrees := make([]*model.ServiceTree, 0, pageSize)
		for i := 0; i < len(scoredList) && i < pageSize; i++ {
			limitedTrees = append(limitedTrees, scoredList[i].tree)
		}
		return limitedTrees
	}

	if len(trees) > pageSize {
		return trees[:pageSize]
	}
	return trees
}

func buildFunctionSearchResults(trees []*model.ServiceTree) []*dto.FunctionSearchResult {
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
			result.Callbacks = tree.Function.Callbacks
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
	return functionResults
}
