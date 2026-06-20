package github

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

const githubMaxPageSize = 100

type GitHubReposReq struct {
	Type              string `json:"type" form:"type" widget:"name:仓库范围;type:select;options:全部,自己,成员;options_colors:409EFF,67C23A,E6A23C;render_default:全部"`
	query.PageSortReq `widget:"-"`
}

type GitHubRepo struct {
	ID              int64  `json:"id" widget:"name:仓库ID;type:integer"`
	Name            string `json:"name" widget:"name:仓库名;type:input"`
	FullName        string `json:"full_name" widget:"name:完整名称;type:input"`
	Private         bool   `json:"private" widget:"name:私有;type:switch"`
	Description     string `json:"description" widget:"name:描述;type:text_area"`
	HTMLURL         string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	Language        string `json:"language" widget:"name:语言;type:input"`
	DefaultBranch   string `json:"default_branch" widget:"name:默认分支;type:input"`
	StargazersCount int    `json:"stargazers_count" widget:"name:Stars;type:integer"`
	ForksCount      int    `json:"forks_count" widget:"name:Forks;type:integer"`
	RepoUpdatedAt   string `json:"repo_updated_at" widget:"name:更新时间;type:input"`
	RepoPushedAt    string `json:"repo_pushed_at" widget:"name:推送时间;type:input"`
}

func (r *GitHubRepo) UnmarshalJSON(data []byte) error {
	type githubRepoAlias GitHubRepo
	var raw struct {
		githubRepoAlias
		UpdatedAt string `json:"updated_at"`
		PushedAt  string `json:"pushed_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = GitHubRepo(raw.githubRepoAlias)
	r.RepoUpdatedAt = raw.UpdatedAt
	r.RepoPushedAt = raw.PushedAt
	return nil
}

func GitHubRepos(ctx *app.Context, resp response.Response) error {
	var req GitHubReposReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	pageSize := req.PageSortReq.GetLimit(20)
	if pageSize > githubMaxPageSize {
		pageSize = githubMaxPageSize
		req.PageSortReq.PageSize = githubMaxPageSize
	}
	queryValues := map[string]string{
		"page":      strconv.Itoa(req.PageSortReq.GetPage()),
		"per_page":  strconv.Itoa(pageSize),
		"sort":      "updated",
		"direction": "desc",
		"type":      githubRepoType(req.Type),
	}
	var repos []GitHubRepo
	proxyResp, err := callGitHubJSON(ctx, "/user/repos", queryValues, &repos)
	if err != nil {
		return err
	}
	total := githubEstimatedTotal(proxyResp.Headers, req.PageSortReq.GetPage(), pageSize, len(repos))
	return resp.Table(response.TableResult{
		Items:      repos,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func githubRepoType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "自己", "owner":
		return "owner"
	case "成员", "member":
		return "member"
	default:
		return "all"
	}
}

func githubEstimatedTotal(headers map[string]string, page, pageSize, itemCount int) int64 {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if lastPage := githubLastPage(headers["Link"]); lastPage > 0 {
		return int64((lastPage-1)*pageSize + itemCount)
	}
	return int64((page-1)*pageSize + itemCount)
}

func githubLastPage(linkHeader string) int {
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="last"`) {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start < 0 || end <= start {
			continue
		}
		u, err := url.Parse(part[start+1 : end])
		if err != nil {
			continue
		}
		page, _ := strconv.Atoi(u.Query().Get("page"))
		return page
	}
	return 0
}

var GitHubReposTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 仓库列表",
		Desc:       "使用当前用户已授权的 GitHub 连接器读取 /user/repos 仓库列表。",
		Tags:       []string{"GitHub", "连接器", "仓库"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/repos", "读取当前用户仓库", "repo"),
		},
		Request: &GitHubReposReq{},
	},
	AutoCrudTable: &GitHubRepo{},
}

func init() {
	packageContext.GET("repos.table", GitHubRepos, GitHubReposTemplate)
}
