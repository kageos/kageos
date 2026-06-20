package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
)

func githubEndpoint(method, urlPath, name string, requiredScopes ...string) app.ConnectorEndpoint {
	return app.ConnectorEndpoint{
		Provider:       "github",
		Method:         method,
		URL:            urlPath,
		Name:           name,
		RequiredScopes: requiredScopes,
	}
}

func bindGitHubRepoReq(ctx *app.Context, req interface{}, owner *string, repo *string) (bool, error) {
	if err := ctx.ShouldBindValidate(req); err != nil {
		return false, err
	}
	*owner, *repo = normalizeGitHubRepoSelection(*owner, *repo)
	if *repo == "" {
		return false, nil
	}
	if *owner == "" {
		defaultOwner, err := githubDefaultOwner(ctx)
		if err != nil {
			return false, err
		}
		*owner = defaultOwner
	}
	return true, nil
}

func normalizeGitHubRepoSelection(owner, repo string) (string, string) {
	owner = strings.TrimSpace(owner)
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if repo == "" {
		return owner, ""
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "/")
	}
	return owner, repo
}

func githubDefaultOwner(ctx *app.Context) (string, error) {
	resolved, err := ctx.GetConnector("github")
	if err != nil {
		return "", err
	}
	for _, value := range []string{
		resolved.Connection.DisplayName,
		resolved.Connection.ExternalAccountID,
	} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("未提供 owner，且当前 GitHub 连接未返回可用账号名")
}

func githubRepoOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	items, err := githubRepoFuzzyValueItems(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return &callback.OnSelectFuzzyResp{Items: items}, nil
	}

	keyword := strings.ToLower(strings.TrimSpace(req.Keyword()))
	queryValues := map[string]string{
		"page":      "1",
		"per_page":  "100",
		"sort":      "updated",
		"direction": "desc",
		"type":      "all",
	}
	var repos []GitHubRepo
	if _, err := callGitHubJSON(ctx, "/user/repos", queryValues, &repos); err != nil {
		return nil, err
	}
	items = make([]*callback.SelectFuzzyItem, 0, 20)
	for _, repo := range repos {
		fullName := strings.TrimSpace(repo.FullName)
		name := strings.TrimSpace(repo.Name)
		desc := strings.TrimSpace(repo.Description)
		if fullName == "" {
			fullName = name
		}
		if keyword != "" && !strings.Contains(strings.ToLower(fullName), keyword) && !strings.Contains(strings.ToLower(desc), keyword) {
			continue
		}
		items = append(items, githubRepoSelectItem(repo))
		if len(items) >= 20 {
			break
		}
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

func githubRepoFuzzyValueItems(ctx *app.Context, req *callback.OnSelectFuzzyReq) ([]*callback.SelectFuzzyItem, error) {
	if req == nil || req.IsByKeyword() {
		return nil, nil
	}
	values := make([]string, 0)
	switch value := req.GetValue().(type) {
	case string:
		values = append(values, value)
	case []string:
		values = append(values, value...)
	case []interface{}:
		for _, item := range value {
			values = append(values, fmt.Sprintf("%v", item))
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(values))
	for _, value := range values {
		fullName := strings.Trim(strings.TrimSpace(value), "/")
		if fullName == "" {
			continue
		}
		owner, repoName := normalizeGitHubRepoSelection("", fullName)
		if owner != "" && repoName != "" {
			apiPath, err := githubRepoAPIPath(owner, repoName)
			if err == nil {
				var repo GitHubRepo
				if _, callErr := callGitHubJSON(ctx, apiPath, nil, &repo); callErr == nil {
					items = append(items, githubRepoSelectItem(repo))
					continue
				}
			}
		}
		items = append(items, &callback.SelectFuzzyItem{
			Value: fullName,
			Label: fullName,
		})
	}
	return items, nil
}

func githubRepoSelectItem(repo GitHubRepo) *callback.SelectFuzzyItem {
	fullName := strings.TrimSpace(repo.FullName)
	if fullName == "" {
		fullName = strings.TrimSpace(repo.Name)
	}
	display := map[string]interface{}{
		"仓库":    fullName,
		"语言":    repo.Language,
		"Stars": repo.StargazersCount,
		"Forks": repo.ForksCount,
		"私有":    repo.Private,
	}
	if strings.TrimSpace(repo.Description) != "" {
		display["描述"] = strings.TrimSpace(repo.Description)
	}
	return &callback.SelectFuzzyItem{
		Value:       fullName,
		Label:       fullName,
		DisplayInfo: display,
	}
}

func callGitHubJSON(ctx *app.Context, path string, query map[string]string, out interface{}) (*app.ConnectorResponse, error) {
	proxyResp, err := ctx.CallConnector("github", app.ConnectorRequest{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return proxyResp, nil
	}
	if len(proxyResp.Body) == 0 {
		return nil, fmt.Errorf("GitHub API 响应为空")
	}
	if err := json.Unmarshal(proxyResp.Body, out); err != nil {
		return nil, fmt.Errorf("解析 GitHub API 响应失败: %w", err)
	}
	return proxyResp, nil
}

func githubRepoAPIPath(owner, repo string, segments ...string) (string, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return "", fmt.Errorf("owner 和 repo 不能为空")
	}
	parts := []string{"repos", owner, repo}
	parts = append(parts, segments...)
	for i, part := range parts {
		parts[i] = url.PathEscape(strings.Trim(part, "/"))
	}
	return "/" + strings.Join(parts, "/"), nil
}

func githubPagedQuery(pageInfo *query.PageSortReq, extra map[string]string) (map[string]string, int) {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	pageSize := pageInfo.GetLimit(20)
	if pageSize > githubMaxPageSize {
		pageSize = githubMaxPageSize
		pageInfo.PageSize = githubMaxPageSize
	}
	values := map[string]string{
		"page":     strconv.Itoa(pageInfo.GetPage()),
		"per_page": strconv.Itoa(pageSize),
	}
	for key, value := range extra {
		key = strings.TrimSpace(key)
		if key != "" {
			values[key] = value
		}
	}
	return values, pageSize
}

func githubTableTotal(proxyResp *app.ConnectorResponse, pageInfo *query.PageSortReq, pageSize, itemCount int) int64 {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	headers := map[string]string(nil)
	if proxyResp != nil {
		headers = proxyResp.Headers
	}
	return githubEstimatedTotal(headers, pageInfo.GetPage(), pageSize, itemCount)
}

func githubPrettyJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}

func parseGitHubQueryText(text string) (map[string]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("Query JSON 解析失败: %w", err)
	}
	values := make(map[string]string, len(raw))
	for key, value := range raw {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			values[key] = typed
		case float64:
			values[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			values[key] = strconv.FormatBool(typed)
		default:
			data, _ := json.Marshal(typed)
			values[key] = string(data)
		}
	}
	return values, nil
}
