package service

import (
	"sort"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
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

func normalizeSearchResourcesPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
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

func calculateSearchResourcesFetchSize(page, pageSize int, keyword string) int {
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
	pathLower := strings.ToLower(strings.TrimSpace(tree.FullCodePath))
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

		if strings.Contains(pathLower, k) {
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

func searchResourcesRelevanceScore(tree *model.ServiceTree, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}

	score := 0
	nameLower := strings.ToLower(strings.TrimSpace(tree.Name))
	codeLower := strings.ToLower(strings.TrimSpace(tree.Code))
	pathLower := strings.ToLower(strings.TrimSpace(tree.FullCodePath))
	descLower := strings.ToLower(tree.Description)
	tagsLower := strings.ToLower(tree.Tags)
	tagSlice := tree.GetTagsSlice()

	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}

		if nameLower == k {
			score += 12
		} else if strings.Contains(nameLower, k) {
			score += 4
		}

		if codeLower == k {
			score += 8
		} else if strings.Contains(codeLower, k) {
			score += 3
		}

		if strings.Contains(pathLower, k) {
			score += 3
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

	if tree.Type == model.ServiceTreeTypePackage {
		score += 1
	}

	return score
}

func searchResourcesMatchTier(tree *model.ServiceTree, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}

	tier := 0
	nameLower := strings.ToLower(strings.TrimSpace(tree.Name))
	codeLower := strings.ToLower(strings.TrimSpace(tree.Code))
	pathLower := strings.ToLower(strings.TrimSpace(tree.FullCodePath))
	pathTailLower := strings.ToLower(strings.TrimSpace(getSearchPathTail(tree.FullCodePath)))
	descLower := strings.ToLower(tree.Description)
	tagsLower := strings.ToLower(tree.Tags)
	tagSlice := tree.GetTagsSlice()

	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}

		switch {
		case nameLower == k:
			tier = maxSearchTier(tier, 6)
		case codeLower == k || pathTailLower == k:
			tier = maxSearchTier(tier, 5)
		case strings.Contains(nameLower, k):
			tier = maxSearchTier(tier, 4)
		case strings.Contains(codeLower, k) || strings.Contains(pathLower, k):
			tier = maxSearchTier(tier, 3)
		case hasExactMatchedTag(tagSlice, k):
			tier = maxSearchTier(tier, 3)
		case strings.Contains(descLower, k) || strings.Contains(tagsLower, k):
			tier = maxSearchTier(tier, 2)
		}
	}

	return tier
}

func getSearchPathTail(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func hasExactMatchedTag(tags []string, keyword string) bool {
	for _, t := range tags {
		if strings.ToLower(strings.TrimSpace(t)) == keyword {
			return true
		}
	}
	return false
}

func maxSearchTier(current, candidate int) int {
	if candidate > current {
		return candidate
	}
	return current
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

func rankAndLimitSearchResources(
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
			tier  int
		}

		scoredList := make([]scored, 0, len(trees))
		for _, t := range trees {
			scoredList = append(scoredList, scored{
				tree:  t,
				score: searchResourcesRelevanceScore(t, keywords),
				tier:  searchResourcesMatchTier(t, keywords),
			})
		}
		sort.Slice(scoredList, func(i, j int) bool {
			if scoredList[i].tier != scoredList[j].tier {
				return scoredList[i].tier > scoredList[j].tier
			}
			if scoredList[i].tree.RunCount != scoredList[j].tree.RunCount {
				return scoredList[i].tree.RunCount > scoredList[j].tree.RunCount
			}
			if scoredList[i].score != scoredList[j].score {
				return scoredList[i].score > scoredList[j].score
			}
			return scoredList[i].tree.UpdatedAt.GetUnix() > scoredList[j].tree.UpdatedAt.GetUnix()
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
			result.Callbacks = tree.Function.GetCallbacks()
			if schema, err := functionschema.Parse(tree.Function.Schema); err == nil {
				result.Schema = schema
			}
		}
		functionResults = append(functionResults, result)
	}
	return functionResults
}

func buildResourceSearchResults(trees []*model.ServiceTree, keyword string) []*dto.ResourceSearchResult {
	results := make([]*dto.ResourceSearchResult, 0, len(trees))
	for _, tree := range trees {
		description := buildResourceDescription(tree)
		result := &dto.ResourceSearchResult{
			ID:           tree.ID,
			Name:         tree.Name,
			Code:         tree.Code,
			Type:         tree.Type,
			Description:  description,
			Tags:         tree.Tags,
			TemplateType: tree.TemplateType,
			FullCodePath: tree.FullCodePath,
			RunCount:     tree.RunCount,
			MatchSource:  detectResourceMatchSource(tree, keyword),
			Snippet:      buildResourceSnippet(tree, keyword, description),
		}
		if tree.App != nil {
			result.AppID = tree.AppID
			result.AppUser = tree.App.User
			result.AppCode = tree.App.Code
		}
		results = append(results, result)
	}
	return results
}

func buildResourceDescription(tree *model.ServiceTree) string {
	if tree.Type != model.ServiceTreeTypeDocs {
		return tree.Description
	}

	if strings.TrimSpace(tree.SearchDocSummary) != "" {
		return limitResourcePreviewText(tree.SearchDocSummary, 220)
	}
	if strings.TrimSpace(tree.Description) != "" {
		return limitResourcePreviewText(tree.Description, 220)
	}
	return ""
}

func detectResourceMatchSource(tree *model.ServiceTree, keyword string) string {
	if tree.Type == model.ServiceTreeTypeDocs {
		keywords := splitSearchKeywordsForRelevance(keyword)
		descLower := strings.ToLower(tree.Description)
		summaryLower := strings.ToLower(tree.SearchDocSummary)
		for _, k := range keywords {
			k = strings.ToLower(strings.TrimSpace(k))
			if k != "" && (strings.Contains(descLower, k) || strings.Contains(summaryLower, k)) {
				return "doc"
			}
		}
	}
	return "node"
}

func buildResourceSnippet(tree *model.ServiceTree, keyword string, description string) string {
	if tree.Type == model.ServiceTreeTypeDocs {
		if strings.TrimSpace(description) != "" {
			return description
		}
		return tree.FullCodePath
	}

	if strings.TrimSpace(description) != "" {
		return trimSearchSnippet(description, keyword, 120)
	}
	if strings.TrimSpace(tree.Tags) != "" {
		return trimSearchSnippet(tree.Tags, keyword, 120)
	}
	return tree.FullCodePath
}

func limitResourcePreviewText(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" || limit <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func trimSearchSnippet(text, keyword string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	keywords := splitSearchKeywordsForRelevance(keyword)
	start := 0
	lowerText := strings.ToLower(text)
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}
		if idx := strings.Index(lowerText, k); idx >= 0 {
			start = len([]rune(text[:idx])) - limit/3
			if start < 0 {
				start = 0
			}
			break
		}
	}

	end := start + limit
	if end > len(runes) {
		end = len(runes)
		start = end - limit
		if start < 0 {
			start = 0
		}
	}

	prefix := ""
	if start > 0 {
		prefix = "..."
	}
	suffix := ""
	if end < len(runes) {
		suffix = "..."
	}
	return prefix + string(runes[start:end]) + suffix
}
