package github

import (
	"encoding/json"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type GitHubAccountListReq struct {
	query.PageSortReq `widget:"-"`
}

type GitHubOrg struct {
	Login       string `json:"login" widget:"name:组织;type:input"`
	ID          int64  `json:"id" widget:"name:组织ID;type:integer"`
	Description string `json:"description" widget:"name:描述;type:text_area"`
	HTMLURL     string `json:"html_url" widget:"name:主页;type:link;target:_blank;link_type:primary"`
}

type GitHubUserBrief struct {
	Login   string `json:"login" widget:"name:登录名;type:input"`
	ID      int64  `json:"id" widget:"name:账号ID;type:integer"`
	Type    string `json:"type" widget:"name:类型;type:input"`
	HTMLURL string `json:"html_url" widget:"name:主页;type:link;target:_blank;link_type:primary"`
}

type GitHubGist struct {
	ID              string `json:"id" widget:"name:Gist ID;type:input"`
	Description     string `json:"description" widget:"name:描述;type:text_area"`
	Public          bool   `json:"public" widget:"name:公开;type:switch"`
	HTMLURL         string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	Comments        int    `json:"comments" widget:"name:评论数;type:integer"`
	FileCount       int    `json:"file_count" widget:"name:文件数;type:integer"`
	GistCreatedTime string `json:"gist_created_time" widget:"name:创建时间;type:input"`
	GistUpdatedTime string `json:"gist_updated_time" widget:"name:更新时间;type:input"`
}

func (g *GitHubGist) UnmarshalJSON(data []byte) error {
	type gistAlias GitHubGist
	var raw struct {
		gistAlias
		Files       map[string]interface{} `json:"files"`
		CreatedTime string                 `json:"created_at"`
		UpdatedTime string                 `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*g = GitHubGist(raw.gistAlias)
	g.FileCount = len(raw.Files)
	g.GistCreatedTime = raw.CreatedTime
	g.GistUpdatedTime = raw.UpdatedTime
	return nil
}

func GitHubOrgs(ctx *app.Context, resp response.Response) error {
	var req GitHubAccountListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, nil)
	var items []GitHubOrg
	proxyResp, err := callGitHubJSON(ctx, "/user/orgs", queryValues, &items)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].HTMLURL == "" && items[i].Login != "" {
			items[i].HTMLURL = "https://github.com/" + items[i].Login
		}
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func GitHubFollowers(ctx *app.Context, resp response.Response) error {
	return githubUserBriefTable(ctx, resp, "/user/followers")
}

func GitHubFollowing(ctx *app.Context, resp response.Response) error {
	return githubUserBriefTable(ctx, resp, "/user/following")
}

func githubUserBriefTable(ctx *app.Context, resp response.Response, path string) error {
	var req GitHubAccountListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, nil)
	var items []GitHubUserBrief
	proxyResp, err := callGitHubJSON(ctx, path, queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func GitHubStarredRepos(ctx *app.Context, resp response.Response) error {
	var req GitHubAccountListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, map[string]string{"sort": "updated", "direction": "desc"})
	var items []GitHubRepo
	proxyResp, err := callGitHubJSON(ctx, "/user/starred", queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func GitHubGists(ctx *app.Context, resp response.Response) error {
	var req GitHubAccountListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, nil)
	var items []GitHubGist
	proxyResp, err := callGitHubJSON(ctx, "/gists", queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var GitHubOrgsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 组织列表",
		Desc:       "读取当前 GitHub 用户可见的组织列表。",
		Tags:       []string{"GitHub", "组织", "连接器"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/orgs", "读取当前用户组织", "read:org"),
		},
		Request: &GitHubAccountListReq{},
	},
	AutoCrudTable: &GitHubOrg{},
}

var GitHubFollowersTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 关注者",
		Desc:       "读取当前 GitHub 用户的 followers 列表。",
		Tags:       []string{"GitHub", "用户", "关注者"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/followers", "读取关注者", "read:user"),
		},
		Request: &GitHubAccountListReq{},
	},
	AutoCrudTable: &GitHubUserBrief{},
}

var GitHubFollowingTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 正在关注",
		Desc:       "读取当前 GitHub 用户正在关注的账号列表。",
		Tags:       []string{"GitHub", "用户", "关注"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/following", "读取正在关注", "read:user"),
		},
		Request: &GitHubAccountListReq{},
	},
	AutoCrudTable: &GitHubUserBrief{},
}

var GitHubStarredReposTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub Star 仓库",
		Desc:       "读取当前 GitHub 用户 star 过的仓库。",
		Tags:       []string{"GitHub", "仓库", "Star"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/starred", "读取 Star 仓库", "read:user"),
		},
		Request: &GitHubAccountListReq{},
	},
	AutoCrudTable: &GitHubRepo{},
}

var GitHubGistsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub Gist 列表",
		Desc:       "读取当前 GitHub 用户可见的 Gist 列表。",
		Tags:       []string{"GitHub", "Gist", "连接器"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/gists", "读取 Gist 列表", "gist"),
		},
		Request: &GitHubAccountListReq{},
	},
	AutoCrudTable: &GitHubGist{},
}

func init() {
	packageContext.GET("orgs.table", GitHubOrgs, GitHubOrgsTemplate)
	packageContext.GET("followers.table", GitHubFollowers, GitHubFollowersTemplate)
	packageContext.GET("following.table", GitHubFollowing, GitHubFollowingTemplate)
	packageContext.GET("starred_repos.table", GitHubStarredRepos, GitHubStarredReposTemplate)
	packageContext.GET("gists.table", GitHubGists, GitHubGistsTemplate)
}
