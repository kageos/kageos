package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type SearchTool struct{ registry *ToolRegistry }

type searchArgs struct {
	Keyword      string `json:"keyword" schema_desc:"搜索关键词，自然写即可；多个候选关键词可用竖线 | 表示 OR。已知完整路径时优先传 full_code_path"`
	FullCodePath string `json:"full_code_path" schema_desc:"已知完整路径时直接传这里，支持精确资源路径或目录前缀；例如 /system/demos/weixin/wechat_articles/search_articles.form"`
	ResourceType string `json:"resource_type" schema_desc:"搜索类型：all=全部资源，function=可执行函数，directory=目录，docs=文档，tool=内置工具" schema_enum:"all,function,directory,docs,tool"`
	TemplateType string `json:"template_type" schema_desc:"函数类型过滤，仅搜索可执行函数时使用" schema_enum:"form,table,chart"`
	Capability   string `json:"capability" schema_desc:"函数能力过滤：form 支持 submit；chart 支持 query；table 支持 read/create/update/delete/read-only" schema_enum:"read,read-only,create,update,delete,submit,query"`
	Page         *int   `json:"page" schema_desc:"页码，默认 1"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数，默认使用 limit 或 20，最多 100"`
	Limit        *int   `json:"limit" schema_desc:"最多返回条数"`
	SchemaOutput string `json:"schema_output" schema_desc:"函数 schema 输出方式：summary=字段摘要（默认），both=字段摘要和原始 JSON" schema_enum:"summary,both"`
}

var searchToolDef = toolDefinition[searchArgs](
	"search",
	"统一搜索工具：只负责定位和确认资源，不执行、不写入。能搜：工作台目录、工作台文档、可执行函数（Form/Table/Chart）及其 schema、内置工具、/system/prompt 下系统文档/SDK/案例的内容命中行。不能搜：本地源码全文（用 read_file）、文档全文（用 read_doc）、外网内容（用 web_search）。默认搜索当前账号有权限看到的全局资源；权限由平台统一判断，不需要也不要传 scope、directory、user 或 app。已知路径时传 full_code_path，不要把完整路径塞进 keyword；full_code_path 支持精确函数/文档路径，也支持目录前缀。resource_type=function 时会返回函数 schema 摘要；可配合 template_type=form/table/chart 和 capability=read/create/update/delete/submit/query 缩小范围，可用 schema_output=both 查看原始 JSON。执行 run_form_submit/run_table_search/run_table_create/run_table_update/run_table_delete/run_chart_query/run_on_select_fuzzy 前先用 search 确认字段名、必填项、枚举、文件字段和能力。keyword 自然写即可；多个备选关键词可用竖线 | 表示 OR。resource_type=tool 可查内置工具。",
)

func (t *SearchTool) Definition() dto.ToolDef {
	return searchToolDef
}

func (t *SearchTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchArgs](call.Args)
	if err != nil {
		return toolResult("search 参数解析失败: "+err.Error(), true)
	}
	return runSearchTool(ctx, t.registry, args)
}

func splitSearchKeywords(keyword string) []string {
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

type searchRequestOutput string

const (
	searchRequestOutputSummary searchRequestOutput = "summary"
	searchRequestOutputBoth    searchRequestOutput = "both"
)

func normalizeSearchRequestOutput(raw string) searchRequestOutput {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(searchRequestOutputBoth):
		return searchRequestOutputBoth
	default:
		return searchRequestOutputSummary
	}
}

type searchResultData struct {
	Keyword      string                      `json:"keyword"`
	FullCodePath string                      `json:"full_code_path,omitempty"`
	ResourceType string                      `json:"resource_type,omitempty"`
	TemplateType string                      `json:"template_type,omitempty"`
	Capability   string                      `json:"capability,omitempty"`
	Page         int                         `json:"page"`
	PageSize     int                         `json:"page_size"`
	Total        int64                       `json:"total"`
	MatchedTools []dto.ToolDef               `json:"matched_tools,omitempty"`
	Functions    []*dto.FunctionSearchResult `json:"functions,omitempty"`
	Items        []*dto.ResourceSearchResult `json:"items,omitempty"`
	DocMatches   []searchDocMatch            `json:"doc_matches,omitempty"`
}

type searchDocMatch struct {
	Name         string `json:"name,omitempty"`
	FullCodePath string `json:"full_code_path,omitempty"`
	Line         int    `json:"line,omitempty"`
	Snippet      string `json:"snippet,omitempty"`
}

func normalizeSearchPageSize(args searchArgs) int {
	pageSize := 20
	if args.Limit != nil && *args.Limit > 0 {
		pageSize = *args.Limit
	}
	if args.PageSize != nil && *args.PageSize > 0 {
		pageSize = *args.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return pageSize
}

func normalizeSearchPage(args searchArgs) int {
	if args.Page != nil && *args.Page > 0 {
		return *args.Page
	}
	return 1
}

func runSearchTool(ctx context.Context, registry *ToolRegistry, args searchArgs) ToolResult {
	keywordRaw := strings.TrimSpace(args.Keyword)
	fullCodePath := normalizeWorkspacePath(args.FullCodePath)
	resourceType := normalizeSearchResourceType(args.ResourceType)
	templateType := strings.TrimSpace(args.TemplateType)
	capability := normalizeSearchCapability(args.Capability)
	requestOutput := normalizeSearchRequestOutput(args.SchemaOutput)
	page := normalizeSearchPage(args)
	pageSize := normalizeSearchPageSize(args)
	displayKeyword := firstNonEmptyString(keywordRaw, fullCodePath)

	var matchedTools []dto.ToolDef
	if shouldSearchBuiltinTools(resourceType, keywordRaw) {
		matchedTools = searchBuiltinTools(ctx, registry, keywordRaw, page, pageSize)
	}

	functions := make([]*dto.FunctionSearchResult, 0)
	if shouldSearchFunctions(resourceType, templateType, capability, fullCodePath) {
		fetchPage, fetchPageSize := page, pageSize
		if capability != "" || fullCodePath != "" {
			fetchPage = 1
			fetchPageSize = 100
		}
		resp, err := apicall.SearchFunctions(ctx, &dto.SearchFunctionsReq{
			Keyword:      keywordRaw,
			FullCodePath: fullCodePath,
			TemplateType: templateType,
			Page:         fetchPage,
			PageSize:     fetchPageSize,
		})
		if err != nil {
			logger.Warnf(ctx, "[Search] SearchFunctions err: %v", err)
		} else if resp != nil {
			functions = resp.Functions
		}
		if capability != "" {
			functions = filterSearchFunctionsByCapability(functions, capability)
			functions = paginateSearchFunctions(functions, page, pageSize)
		}
	}

	items := make([]*dto.ResourceSearchResult, 0)
	if shouldSearchResources(resourceType) {
		resp, err := apicall.SearchResources(ctx, &dto.SearchResourcesReq{
			Keyword:      keywordRaw,
			FullCodePath: fullCodePath,
			ResourceType: apiSearchResourceType(resourceType),
			Page:         page,
			PageSize:     pageSize,
		})
		if err != nil {
			logger.Warnf(ctx, "[Search] SearchResources err: %v", err)
		} else if resp != nil {
			items = resp.Items
		}
		if resourceType == "all" {
			items = filterSearchResourceItemsByType(items, "function", false)
		}
	}

	docMatches := searchPromptDocMatches(ctx, fullCodePath, keywordRaw, pageSize)

	data := searchResultData{
		Keyword:      displayKeyword,
		FullCodePath: fullCodePath,
		ResourceType: resourceType,
		TemplateType: templateType,
		Capability:   capability,
		Page:         page,
		PageSize:     pageSize,
		MatchedTools: matchedTools,
		Functions:    functions,
		Items:        items,
		DocMatches:   docMatches,
	}
	data.Total = int64(len(matchedTools) + len(functions) + len(items) + len(docMatches))

	if data.Total == 0 {
		message := "未匹配到资源。已知完整路径时传 full_code_path；需要找函数 schema 时用 resource_type=function；多个候选词可用 | 分隔。"
		if displayKeyword == "" {
			message = "未传 keyword 或 full_code_path，当前没有可展示的搜索结果。可以传关键词、完整路径，或 resource_type=tool 查看内置工具。"
		} else if prompt.IsPromptDocPath(fullCodePath) {
			docPath := fullCodePath
			if resolvedPath, _, _ := resolvePromptDocSearchContent(ctx, fullCodePath); resolvedPath != "" {
				docPath = resolvedPath
			}
			message = fmt.Sprintf("未在内置文档/案例 %s 中命中内容。需要完整查看该案例时直接调用 read_doc(directory=%q)；不要用 read_file 读取 /system/prompt 案例路径。", docPath, docPath)
		}
		return toolResultWithData(formatSearchMeta(data)+"\n\n"+message, false, data)
	}

	return toolResultWithData(formatSearchOutput(data, requestOutput), false, data)
}
