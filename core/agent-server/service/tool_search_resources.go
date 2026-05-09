package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type SearchResourcesTool struct{}

type searchResourcesArgs struct {
	Keyword      string `json:"keyword" schema_desc:"搜索关键词，支持竖线分隔多个关键词"`
	ResourceType string `json:"resource_type" schema_desc:"资源类型过滤：all/package/function/docs/board" schema_enum:"all,package,function,docs,board"`
	Scope        string `json:"scope" schema_desc:"搜索范围：visible=当前用户可见资源（默认），system=官方/system 资源，current_user=当前用户下资源，current_app=当前工作区应用内资源" schema_enum:"visible,system,current_user,current_app"`
	User         string `json:"user" schema_desc:"按用户名过滤；传入后优先于 scope 推导"`
	App          string `json:"app" schema_desc:"按应用 code 过滤；通常和 user 搭配使用"`
	Page         *int   `json:"page" schema_desc:"页码，默认 1"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数，默认 20，最多 100"`
}

type searchResourcesResultData struct {
	Keyword      string                      `json:"keyword"`
	ResourceType string                      `json:"resource_type,omitempty"`
	Scope        string                      `json:"scope"`
	User         string                      `json:"user,omitempty"`
	App          string                      `json:"app,omitempty"`
	Page         int                         `json:"page"`
	PageSize     int                         `json:"page_size"`
	Total        int64                       `json:"total"`
	Items        []*dto.ResourceSearchResult `json:"items,omitempty"`
}

var searchResourcesToolDef = toolDefinition[searchResourcesArgs](
	"search_resources",
	"搜索服务树资源：目录、函数、文档和讨论区。默认 scope=visible 搜当前用户可见资源；可用 scope=system 搜官方/system 资源，scope=current_app 搜当前工作区应用，或传 user/app 精确过滤。适合先找应用目录、文档、函数位置；若要执行函数，再用 search_tools 获取 schema 摘要后调用 run_form_submit/run_table_search/run_chart_query。",
)

func (t *SearchResourcesTool) Definition() dto.ToolDef {
	return searchResourcesToolDef
}

func (t *SearchResourcesTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchResourcesArgs](call.Args)
	if err != nil {
		return toolResult("search_resources 参数解析失败: "+err.Error(), true)
	}
	return runSearchResourcesTool(ctx, args, call.FullCodePath)
}

func runSearchResourcesTool(ctx context.Context, args searchResourcesArgs, currentFullCodePath string) ToolResult {
	keyword := strings.TrimSpace(args.Keyword)
	resourceType := normalizeSearchResourcesType(args.ResourceType)
	page := 1
	if args.Page != nil && *args.Page > 0 {
		page = *args.Page
	}
	pageSize := 20
	if args.PageSize != nil && *args.PageSize > 0 {
		pageSize = *args.PageSize
		if pageSize > 100 {
			pageSize = 100
		}
	}
	user, app, scope := resolveSearchScopeUserApp(args.Scope, args.User, args.App, currentFullCodePath, searchScopeVisible)

	resp, err := apicall.SearchResources(ctx, &dto.SearchResourcesReq{
		User:         user,
		App:          app,
		Keyword:      keyword,
		ResourceType: resourceType,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		logger.Warnf(ctx, "[SearchResources] SearchResources err: %v", err)
		return toolResult("search_resources 调用失败: "+err.Error(), true)
	}

	data := searchResourcesResultData{
		Keyword:      keyword,
		ResourceType: resourceType,
		Scope:        scope,
		User:         user,
		App:          app,
		Page:         page,
		PageSize:     pageSize,
	}
	if resp != nil {
		data.Total = resp.Total
		data.Items = resp.Items
	}

	if len(data.Items) == 0 {
		return toolResultWithData(formatSearchResourcesMeta(data)+"\n\n未匹配到服务树资源。可调整 keyword、resource_type 或 scope 再试。", false, data)
	}
	return toolResultWithData(formatSearchResourcesOutput(data), false, data)
}

func normalizeSearchResourcesType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return "all"
	case "package", "function", "docs", "board":
		return strings.ToLower(strings.TrimSpace(raw))
	case "doc", "document":
		return "docs"
	default:
		return "all"
	}
}

func formatSearchResourcesMeta(data searchResourcesResultData) string {
	parts := []string{
		"范围=" + data.Scope,
		fmt.Sprintf("page=%d", data.Page),
		fmt.Sprintf("page_size=%d", data.PageSize),
	}
	if data.User != "" {
		parts = append(parts, "user="+data.User)
	}
	if data.App != "" {
		parts = append(parts, "app="+data.App)
	}
	if data.ResourceType != "" && data.ResourceType != "all" {
		parts = append(parts, "resource_type="+data.ResourceType)
	}
	if data.Total > 0 {
		parts = append(parts, fmt.Sprintf("total=%d", data.Total))
	}
	return "搜索参数：" + strings.Join(parts, " | ")
}

func formatSearchResourcesOutput(data searchResourcesResultData) string {
	var b strings.Builder
	if data.Keyword == "" {
		b.WriteString("服务树搜索结果：未传 keyword，返回当前范围内资源。\n")
	} else {
		b.WriteString(fmt.Sprintf("服务树搜索结果：关键词「%s」\n", data.Keyword))
	}
	b.WriteString(formatSearchResourcesMeta(data))
	b.WriteString("\n\n")
	for i, item := range data.Items {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Name))
		b.WriteString("   type: ")
		b.WriteString(item.Type)
		if item.TemplateType != "" {
			b.WriteString(" / ")
			b.WriteString(item.TemplateType)
		}
		b.WriteString("\n")
		b.WriteString("   full_code_path: ")
		b.WriteString(item.FullCodePath)
		b.WriteString("\n")
		if item.AppUser != "" || item.AppCode != "" {
			b.WriteString(fmt.Sprintf("   app: %s/%s\n", item.AppUser, item.AppCode))
		}
		if item.MatchSource != "" {
			b.WriteString("   match_source: ")
			b.WriteString(item.MatchSource)
			b.WriteString("\n")
		}
		if item.Description != "" {
			b.WriteString("   description: ")
			b.WriteString(item.Description)
			b.WriteString("\n")
		}
		if item.Snippet != "" && item.Snippet != item.Description {
			b.WriteString("   snippet: ")
			b.WriteString(item.Snippet)
			b.WriteString("\n")
		}
		if item.RunCount > 0 {
			b.WriteString(fmt.Sprintf("   已使用 %d 次\n", item.RunCount))
		}
	}
	return strings.TrimSpace(b.String())
}
