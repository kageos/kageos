package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type GitHubRepoReq struct {
	Owner string `json:"owner" form:"owner" widget:"name:Owner;type:input;placeholder:默认当前连接的 GitHub 用户"`
	Repo  string `json:"repo" form:"repo" widget:"name:仓库;type:select;placeholder:搜索并选择仓库，或输入 owner/repo" callback:"OnSelectFuzzy"`
}

type GitHubRepoListReq struct {
	Owner             string `json:"owner" form:"owner" widget:"name:Owner;type:input;placeholder:默认当前连接的 GitHub 用户"`
	Repo              string `json:"repo" form:"repo" widget:"name:仓库;type:select;placeholder:搜索并选择仓库，或输入 owner/repo" callback:"OnSelectFuzzy"`
	State             string `json:"state" form:"state" widget:"name:状态;type:select;options:all,open,closed;render_default:open"`
	query.PageSortReq `widget:"-"`
}

type GitHubRepoPageReq struct {
	Owner             string `json:"owner" form:"owner" widget:"name:Owner;type:input;placeholder:默认当前连接的 GitHub 用户"`
	Repo              string `json:"repo" form:"repo" widget:"name:仓库;type:select;placeholder:搜索并选择仓库，或输入 owner/repo" callback:"OnSelectFuzzy"`
	query.PageSortReq `widget:"-"`
}

type GitHubRepoRefReq struct {
	Owner             string `json:"owner" form:"owner" widget:"name:Owner;type:input;placeholder:默认当前连接的 GitHub 用户"`
	Repo              string `json:"repo" form:"repo" widget:"name:仓库;type:select;placeholder:搜索并选择仓库，或输入 owner/repo" callback:"OnSelectFuzzy"`
	Ref               string `json:"ref" form:"ref" widget:"name:分支/引用;type:input;placeholder:main"`
	query.PageSortReq `widget:"-"`
}

type GitHubContentsReq struct {
	Owner             string `json:"owner" form:"owner" widget:"name:Owner;type:input;placeholder:默认当前连接的 GitHub 用户"`
	Repo              string `json:"repo" form:"repo" widget:"name:仓库;type:select;placeholder:搜索并选择仓库，或输入 owner/repo" callback:"OnSelectFuzzy"`
	Path              string `json:"path" form:"path" widget:"name:路径;type:input;placeholder:/"`
	Ref               string `json:"ref" form:"ref" widget:"name:分支/引用;type:input;placeholder:main"`
	query.PageSortReq `widget:"-"`
}

type GitHubBranch struct {
	Name      string `json:"name" widget:"name:分支;type:input"`
	CommitSHA string `json:"commit_sha" widget:"name:Commit SHA;type:input"`
	Protected bool   `json:"protected" widget:"name:受保护;type:switch"`
}

func (b *GitHubBranch) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name      string `json:"name"`
		Protected bool   `json:"protected"`
		Commit    struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Name = raw.Name
	b.Protected = raw.Protected
	b.CommitSHA = raw.Commit.SHA
	return nil
}

type GitHubCommit struct {
	SHA            string `json:"sha" widget:"name:SHA;type:input"`
	Message        string `json:"message" widget:"name:提交信息;type:text_area"`
	AuthorName     string `json:"author_name" widget:"name:作者;type:input"`
	AuthorLogin    string `json:"author_login" widget:"name:GitHub 用户;type:input"`
	CommitTime     string `json:"commit_time" widget:"name:提交时间;type:input"`
	HTMLURL        string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	CommentCount   int    `json:"comment_count" widget:"name:评论数;type:integer"`
	Verified       bool   `json:"verified" widget:"name:签名验证;type:switch"`
	VerifiedReason string `json:"verified_reason" widget:"name:验证原因;type:input"`
}

func (c *GitHubCommit) UnmarshalJSON(data []byte) error {
	var raw struct {
		SHA     string `json:"sha"`
		HTMLURL string `json:"html_url"`
		Author  struct {
			Login string `json:"login"`
		} `json:"author"`
		Commit struct {
			Message      string `json:"message"`
			CommentCount int    `json:"comment_count"`
			Author       struct {
				Name string `json:"name"`
				Date string `json:"date"`
			} `json:"author"`
			Verification struct {
				Verified bool   `json:"verified"`
				Reason   string `json:"reason"`
			} `json:"verification"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.SHA = raw.SHA
	c.HTMLURL = raw.HTMLURL
	c.AuthorLogin = raw.Author.Login
	c.Message = raw.Commit.Message
	c.CommentCount = raw.Commit.CommentCount
	c.AuthorName = raw.Commit.Author.Name
	c.CommitTime = raw.Commit.Author.Date
	c.Verified = raw.Commit.Verification.Verified
	c.VerifiedReason = raw.Commit.Verification.Reason
	return nil
}

type GitHubIssue struct {
	Number           int    `json:"number" widget:"name:编号;type:integer"`
	Title            string `json:"title" widget:"name:标题;type:input"`
	State            string `json:"state" widget:"name:状态;type:select;options:open,closed;options_colors:67C23A,909399"`
	UserLogin        string `json:"user_login" widget:"name:创建人;type:input"`
	LabelsText       string `json:"labels_text" widget:"name:标签;type:input"`
	Comments         int    `json:"comments" widget:"name:评论数;type:integer"`
	IsPullRequest    bool   `json:"is_pull_request" widget:"name:是 PR;type:switch"`
	HTMLURL          string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	IssueCreatedTime string `json:"issue_created_time" widget:"name:创建时间;type:input"`
	IssueUpdatedTime string `json:"issue_updated_time" widget:"name:更新时间;type:input"`
}

func (i *GitHubIssue) UnmarshalJSON(data []byte) error {
	var raw struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		State       string `json:"state"`
		Comments    int    `json:"comments"`
		HTMLURL     string `json:"html_url"`
		CreatedTime string `json:"created_at"`
		UpdatedTime string `json:"updated_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		PullRequest *struct{} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	labels := make([]string, 0, len(raw.Labels))
	for _, label := range raw.Labels {
		if strings.TrimSpace(label.Name) != "" {
			labels = append(labels, label.Name)
		}
	}
	i.Number = raw.Number
	i.Title = raw.Title
	i.State = raw.State
	i.Comments = raw.Comments
	i.HTMLURL = raw.HTMLURL
	i.IssueCreatedTime = raw.CreatedTime
	i.IssueUpdatedTime = raw.UpdatedTime
	i.UserLogin = raw.User.Login
	i.LabelsText = strings.Join(labels, ", ")
	i.IsPullRequest = raw.PullRequest != nil
	return nil
}

type GitHubPullRequest struct {
	Number        int    `json:"number" widget:"name:编号;type:integer"`
	Title         string `json:"title" widget:"name:标题;type:input"`
	State         string `json:"state" widget:"name:状态;type:select;options:open,closed;options_colors:67C23A,909399"`
	UserLogin     string `json:"user_login" widget:"name:创建人;type:input"`
	HeadRef       string `json:"head_ref" widget:"name:Head;type:input"`
	BaseRef       string `json:"base_ref" widget:"name:Base;type:input"`
	Draft         bool   `json:"draft" widget:"name:草稿;type:switch"`
	HTMLURL       string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	PRCreatedTime string `json:"pr_created_time" widget:"name:创建时间;type:input"`
	PRUpdatedTime string `json:"pr_updated_time" widget:"name:更新时间;type:input"`
}

func (p *GitHubPullRequest) UnmarshalJSON(data []byte) error {
	type prAlias GitHubPullRequest
	var raw struct {
		prAlias
		CreatedTime string `json:"created_at"`
		UpdatedTime string `json:"updated_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = GitHubPullRequest(raw.prAlias)
	p.UserLogin = raw.User.Login
	p.HeadRef = raw.Head.Ref
	p.BaseRef = raw.Base.Ref
	p.PRCreatedTime = raw.CreatedTime
	p.PRUpdatedTime = raw.UpdatedTime
	return nil
}

type GitHubRelease struct {
	ID                 int64  `json:"id" widget:"name:Release ID;type:integer"`
	TagName            string `json:"tag_name" widget:"name:Tag;type:input"`
	Name               string `json:"name" widget:"name:名称;type:input"`
	Draft              bool   `json:"draft" widget:"name:草稿;type:switch"`
	Prerelease         bool   `json:"prerelease" widget:"name:预发布;type:switch"`
	AuthorLogin        string `json:"author_login" widget:"name:作者;type:input"`
	HTMLURL            string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	ReleaseCreatedTime string `json:"release_created_time" widget:"name:创建时间;type:input"`
	PublishedTime      string `json:"published_time" widget:"name:发布时间;type:input"`
}

func (r *GitHubRelease) UnmarshalJSON(data []byte) error {
	type releaseAlias GitHubRelease
	var raw struct {
		releaseAlias
		CreatedTime   string `json:"created_at"`
		PublishedTime string `json:"published_at"`
		Author        struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = GitHubRelease(raw.releaseAlias)
	r.ReleaseCreatedTime = raw.CreatedTime
	r.PublishedTime = raw.PublishedTime
	r.AuthorLogin = raw.Author.Login
	return nil
}

type GitHubWorkflow struct {
	ID              int64  `json:"id" widget:"name:Workflow ID;type:integer"`
	Name            string `json:"name" widget:"name:名称;type:input"`
	Path            string `json:"path" widget:"name:路径;type:input"`
	State           string `json:"state" widget:"name:状态;type:input"`
	HTMLURL         string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	BadgeURL        string `json:"badge_url" widget:"name:Badge;type:link;target:_blank;link_type:info"`
	WorkflowCreated string `json:"workflow_created_time" widget:"name:创建时间;type:input"`
	WorkflowUpdated string `json:"workflow_updated_time" widget:"name:更新时间;type:input"`
}

func (w *GitHubWorkflow) UnmarshalJSON(data []byte) error {
	type workflowAlias GitHubWorkflow
	var raw struct {
		workflowAlias
		CreatedTime string `json:"created_at"`
		UpdatedTime string `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*w = GitHubWorkflow(raw.workflowAlias)
	w.WorkflowCreated = raw.CreatedTime
	w.WorkflowUpdated = raw.UpdatedTime
	return nil
}

type GitHubWorkflowRun struct {
	ID             int64  `json:"id" widget:"name:Run ID;type:integer"`
	Name           string `json:"name" widget:"name:名称;type:input"`
	Status         string `json:"status" widget:"name:状态;type:input"`
	Conclusion     string `json:"conclusion" widget:"name:结论;type:input"`
	Event          string `json:"event" widget:"name:事件;type:input"`
	Branch         string `json:"branch" widget:"name:分支;type:input"`
	CommitSHA      string `json:"commit_sha" widget:"name:Commit SHA;type:input"`
	HTMLURL        string `json:"html_url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	RunCreatedTime string `json:"run_created_time" widget:"name:创建时间;type:input"`
	RunUpdatedTime string `json:"run_updated_time" widget:"name:更新时间;type:input"`
}

func (r *GitHubWorkflowRun) UnmarshalJSON(data []byte) error {
	type runAlias GitHubWorkflowRun
	var raw struct {
		runAlias
		HeadBranch  string `json:"head_branch"`
		HeadSHA     string `json:"head_sha"`
		CreatedTime string `json:"created_at"`
		UpdatedTime string `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = GitHubWorkflowRun(raw.runAlias)
	r.Branch = raw.HeadBranch
	r.CommitSHA = raw.HeadSHA
	r.RunCreatedTime = raw.CreatedTime
	r.RunUpdatedTime = raw.UpdatedTime
	return nil
}

type GitHubContentItem struct {
	Name        string `json:"name" widget:"name:名称;type:input"`
	Path        string `json:"path" widget:"name:路径;type:input"`
	Type        string `json:"type" widget:"name:类型;type:select;options:file,dir,symlink,submodule;options_colors:409EFF,67C23A,E6A23C,909399"`
	Size        int64  `json:"size" widget:"name:大小;type:integer"`
	SHA         string `json:"sha" widget:"name:SHA;type:input"`
	HTMLURL     string `json:"html_url" widget:"name:GitHub 链接;type:link;target:_blank;link_type:primary"`
	DownloadURL string `json:"download_url" widget:"name:下载链接;type:link;target:_blank;link_type:info"`
}

type GitHubLanguagesResp struct {
	Owner         string `json:"owner" widget:"name:Owner;type:input"`
	Repo          string `json:"repo" widget:"name:Repo;type:input"`
	LanguagesText string `json:"languages_text" widget:"name:语言字节统计;type:text_area"`
}

type GitHubContributor struct {
	Login         string `json:"login" widget:"name:登录名;type:input"`
	ID            int64  `json:"id" widget:"name:账号ID;type:integer"`
	Contributions int    `json:"contributions" widget:"name:贡献数;type:integer"`
	Type          string `json:"type" widget:"name:类型;type:input"`
	HTMLURL       string `json:"html_url" widget:"name:主页;type:link;target:_blank;link_type:primary"`
}

type GitHubTag struct {
	Name      string `json:"name" widget:"name:Tag;type:input"`
	CommitSHA string `json:"commit_sha" widget:"name:Commit SHA;type:input"`
	Zipball   string `json:"zipball_url" widget:"name:Zip;type:link;target:_blank;link_type:info"`
	Tarball   string `json:"tarball_url" widget:"name:Tar;type:link;target:_blank;link_type:info"`
}

func (t *GitHubTag) UnmarshalJSON(data []byte) error {
	type tagAlias GitHubTag
	var raw struct {
		tagAlias
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = GitHubTag(raw.tagAlias)
	t.CommitSHA = raw.Commit.SHA
	return nil
}

func GitHubRepoDetail(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return resp.Form(&GitHubRepo{Description: "请选择仓库后再查询。"}).Build()
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo)
	if err != nil {
		return err
	}
	var item GitHubRepo
	if _, err := callGitHubJSON(ctx, apiPath, nil, &item); err != nil {
		return err
	}
	return resp.Form(&item).Build()
}

func GitHubRepoBranches(ctx *app.Context, resp response.Response) error {
	return githubRepoSimpleTable[GitHubBranch](ctx, resp, "branches", nil)
}

func GitHubRepoCommits(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoRefReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubCommit](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "commits")
	if err != nil {
		return err
	}
	extra := map[string]string(nil)
	if strings.TrimSpace(req.Ref) != "" {
		extra = map[string]string{"sha": strings.TrimSpace(req.Ref)}
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, extra)
	var items []GitHubCommit
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)), PageInfo: &req.PageSortReq}).Build()
}

func GitHubRepoIssues(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoListReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubIssue](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "issues")
	if err != nil {
		return err
	}
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "open"
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, map[string]string{"state": state, "sort": "updated", "direction": "desc"})
	var items []GitHubIssue
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)), PageInfo: &req.PageSortReq}).Build()
}

func GitHubRepoPulls(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoListReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubPullRequest](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "pulls")
	if err != nil {
		return err
	}
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "open"
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, map[string]string{"state": state, "sort": "updated", "direction": "desc"})
	var items []GitHubPullRequest
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)), PageInfo: &req.PageSortReq}).Build()
}

func GitHubRepoReleases(ctx *app.Context, resp response.Response) error {
	return githubRepoSimpleTable[GitHubRelease](ctx, resp, "releases", nil)
}

func GitHubRepoContributors(ctx *app.Context, resp response.Response) error {
	return githubRepoSimpleTable[GitHubContributor](ctx, resp, "contributors", nil)
}

func GitHubRepoTags(ctx *app.Context, resp response.Response) error {
	return githubRepoSimpleTable[GitHubTag](ctx, resp, "tags", nil)
}

func GitHubRepoWorkflows(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoPageReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubWorkflow](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "actions", "workflows")
	if err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, nil)
	var payload struct {
		TotalCount int64            `json:"total_count"`
		Workflows  []GitHubWorkflow `json:"workflows"`
	}
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &payload)
	if err != nil {
		return err
	}
	total := payload.TotalCount
	if total <= 0 {
		total = githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(payload.Workflows))
	}
	return resp.Table(response.TableResult{Items: payload.Workflows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

func GitHubRepoWorkflowRuns(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoRefReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubWorkflowRun](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "actions", "runs")
	if err != nil {
		return err
	}
	extra := map[string]string(nil)
	if strings.TrimSpace(req.Ref) != "" {
		extra = map[string]string{"branch": strings.TrimSpace(req.Ref)}
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, extra)
	var payload struct {
		TotalCount   int64               `json:"total_count"`
		WorkflowRuns []GitHubWorkflowRun `json:"workflow_runs"`
	}
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &payload)
	if err != nil {
		return err
	}
	total := payload.TotalCount
	if total <= 0 {
		total = githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(payload.WorkflowRuns))
	}
	return resp.Table(response.TableResult{Items: payload.WorkflowRuns, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

func GitHubRepoContents(ctx *app.Context, resp response.Response) error {
	var req GitHubContentsReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[GitHubContentItem](resp, &req.PageSortReq)
	}
	contentPath := strings.Trim(strings.TrimSpace(req.Path), "/")
	segments := []string{"contents"}
	if contentPath != "" {
		segments = append(segments, strings.Split(contentPath, "/")...)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, segments...)
	if err != nil {
		return err
	}
	extra := map[string]string(nil)
	if strings.TrimSpace(req.Ref) != "" {
		extra = map[string]string{"ref": strings.TrimSpace(req.Ref)}
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, extra)
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, nil)
	if err != nil {
		return err
	}
	var items []GitHubContentItem
	if err := json.Unmarshal(proxyResp.Body, &items); err != nil {
		var item GitHubContentItem
		if oneErr := json.Unmarshal(proxyResp.Body, &item); oneErr != nil {
			return fmt.Errorf("解析 GitHub contents 响应失败: %w", err)
		}
		items = []GitHubContentItem{item}
	}
	total := int64(len(items))
	return resp.Table(response.TableResult{Items: items, TotalCount: total, PageInfo: &query.PageSortReq{Page: req.PageSortReq.GetPage(), PageSize: pageSize}}).Build()
}

func GitHubRepoLanguages(ctx *app.Context, resp response.Response) error {
	var req GitHubRepoReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return resp.Form(&GitHubLanguagesResp{LanguagesText: "请选择仓库后再查询。"}).Build()
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, "languages")
	if err != nil {
		return err
	}
	proxyResp, err := callGitHubJSON(ctx, apiPath, nil, nil)
	if err != nil {
		return err
	}
	return resp.Form(&GitHubLanguagesResp{
		Owner:         req.Owner,
		Repo:          req.Repo,
		LanguagesText: githubPrettyJSON(proxyResp.Body),
	}).Build()
}

func githubRepoSimpleTable[T any](ctx *app.Context, resp response.Response, segment string, extra map[string]string) error {
	var req GitHubRepoPageReq
	ok, err := bindGitHubRepoReq(ctx, &req, &req.Owner, &req.Repo)
	if err != nil {
		return err
	}
	if !ok {
		return githubEmptyTable[T](resp, &req.PageSortReq)
	}
	apiPath, err := githubRepoAPIPath(req.Owner, req.Repo, segment)
	if err != nil {
		return err
	}
	queryValues, pageSize := githubPagedQuery(&req.PageSortReq, extra)
	var items []T
	proxyResp, err := callGitHubJSON(ctx, apiPath, queryValues, &items)
	if err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: githubTableTotal(proxyResp, &req.PageSortReq, pageSize, len(items)), PageInfo: &req.PageSortReq}).Build()
}

func githubEmptyTable[T any](resp response.Response, pageInfo *query.PageSortReq) error {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	return resp.Table(response.TableResult{
		Items:      []T{},
		TotalCount: 0,
		PageInfo:   pageInfo,
	}).Build()
}

var GitHubRepoDetailTemplate = &app.FormTemplate{BaseConfig: app.BaseConfig{
	Name:               "GitHub 仓库详情",
	Desc:               "读取指定仓库详情。",
	Tags:               []string{"GitHub", "仓库", "详情"},
	Connectors:         []string{"github"},
	ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}", "仓库详情", "repo")},
	Request:            &GitHubRepoReq{},
	Response:           &GitHubRepo{},
	OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
		"repo": githubRepoOnSelectFuzzy,
	},
}}

var GitHubRepoBranchesTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库分支",
		Desc:               "读取指定仓库 branches。",
		Tags:               []string{"GitHub", "仓库", "分支"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/branches", "仓库分支", "repo")},
		Request:            &GitHubRepoPageReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubBranch{},
}

var GitHubRepoCommitsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库提交",
		Desc:               "读取指定仓库 commits。",
		Tags:               []string{"GitHub", "仓库", "提交"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/commits", "仓库提交", "repo")},
		Request:            &GitHubRepoRefReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubCommit{},
}

var GitHubRepoIssuesTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库 Issues",
		Desc:               "读取指定仓库 issues。注意 GitHub issues API 也会返回 PR，可用“是 PR”字段区分。",
		Tags:               []string{"GitHub", "仓库", "Issue"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/issues", "仓库 Issues", "repo")},
		Request:            &GitHubRepoListReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubIssue{},
}

var GitHubRepoPullsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库 Pull Requests",
		Desc:               "读取指定仓库 pull requests。",
		Tags:               []string{"GitHub", "仓库", "PR"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/pulls", "仓库 Pull Requests", "repo")},
		Request:            &GitHubRepoListReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubPullRequest{},
}

var GitHubRepoReleasesTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库 Releases",
		Desc:               "读取指定仓库 releases。",
		Tags:               []string{"GitHub", "仓库", "Release"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/releases", "仓库 Releases", "repo")},
		Request:            &GitHubRepoPageReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubRelease{},
}

var GitHubRepoContributorsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库贡献者",
		Desc:               "读取指定仓库 contributors。",
		Tags:               []string{"GitHub", "仓库", "贡献者"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/contributors", "仓库贡献者", "repo")},
		Request:            &GitHubRepoPageReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubContributor{},
}

var GitHubRepoTagsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库 Tags",
		Desc:               "读取指定仓库 tags。",
		Tags:               []string{"GitHub", "仓库", "Tag"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/tags", "仓库 Tags", "repo")},
		Request:            &GitHubRepoPageReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubTag{},
}

var GitHubRepoWorkflowsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub Actions Workflows",
		Desc:               "读取指定仓库 GitHub Actions workflows。",
		Tags:               []string{"GitHub", "Actions", "Workflow"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/actions/workflows", "Actions Workflows", "repo")},
		Request:            &GitHubRepoPageReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubWorkflow{},
}

var GitHubRepoWorkflowRunsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub Actions Runs",
		Desc:               "读取指定仓库 GitHub Actions workflow runs。",
		Tags:               []string{"GitHub", "Actions", "Run"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/actions/runs", "Actions Runs", "repo")},
		Request:            &GitHubRepoRefReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubWorkflowRun{},
}

var GitHubRepoContentsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:               "GitHub 仓库 Contents",
		Desc:               "读取指定仓库目录或文件元数据。",
		Tags:               []string{"GitHub", "仓库", "Contents"},
		Connectors:         []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/contents/{path}", "仓库 Contents", "repo")},
		Request:            &GitHubContentsReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"repo": githubRepoOnSelectFuzzy,
		},
	},
	AutoCrudTable: &GitHubContentItem{},
}

var GitHubRepoLanguagesTemplate = &app.FormTemplate{BaseConfig: app.BaseConfig{
	Name:               "GitHub 仓库语言统计",
	Desc:               "读取指定仓库 languages 字节统计。",
	Tags:               []string{"GitHub", "仓库", "语言"},
	Connectors:         []string{"github"},
	ConnectorEndpoints: []app.ConnectorEndpoint{githubEndpoint("GET", "/repos/{owner}/{repo}/languages", "仓库语言统计", "repo")},
	Request:            &GitHubRepoReq{},
	Response:           &GitHubLanguagesResp{},
	OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
		"repo": githubRepoOnSelectFuzzy,
	},
}}

func init() {
	packageContext.POST("repo_detail.form", GitHubRepoDetail, GitHubRepoDetailTemplate)
	packageContext.GET("repo_branches.table", GitHubRepoBranches, GitHubRepoBranchesTemplate)
	packageContext.GET("repo_commits.table", GitHubRepoCommits, GitHubRepoCommitsTemplate)
	packageContext.GET("repo_issues.table", GitHubRepoIssues, GitHubRepoIssuesTemplate)
	packageContext.GET("repo_pulls.table", GitHubRepoPulls, GitHubRepoPullsTemplate)
	packageContext.GET("repo_releases.table", GitHubRepoReleases, GitHubRepoReleasesTemplate)
	packageContext.GET("repo_contributors.table", GitHubRepoContributors, GitHubRepoContributorsTemplate)
	packageContext.GET("repo_tags.table", GitHubRepoTags, GitHubRepoTagsTemplate)
	packageContext.GET("repo_workflows.table", GitHubRepoWorkflows, GitHubRepoWorkflowsTemplate)
	packageContext.GET("repo_workflow_runs.table", GitHubRepoWorkflowRuns, GitHubRepoWorkflowRunsTemplate)
	packageContext.GET("repo_contents.table", GitHubRepoContents, GitHubRepoContentsTemplate)
	packageContext.POST("repo_languages.form", GitHubRepoLanguages, GitHubRepoLanguagesTemplate)
}
