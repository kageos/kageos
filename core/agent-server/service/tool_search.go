package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/widget"
)

type SearchTool struct{ registry *ToolRegistry }

type searchArgs struct {
	Keyword      string `json:"keyword" schema_desc:"搜索关键词。普通短语可以原样传；多个候选关键词只用竖线 | 分隔表示 OR，不要按空格拆词"`
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
	"统一搜索工具：可以搜工作台目录、文档、可执行函数（Form/Table/Chart）和内置工具。默认搜索当前账号有权限看到的全局资源；权限由平台统一判断，不需要也不要传 scope、directory、user 或 app。已知路径时传 full_code_path，不要把完整路径塞进 keyword；full_code_path 支持精确函数/文档路径，也支持目录前缀。resource_type=function 时会返回函数 schema 摘要，可用 schema_output=both 查看原始 JSON；执行 run_form_submit/run_table_search/run_chart_query 等工具前先用 search 确认字段名、必填项、枚举、文件字段和能力。keyword 普通短语可以原样传；多个备选关键词只用竖线 | 分隔表示 OR，不要按空格拆词。",
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
	}
	data.Total = int64(len(matchedTools) + len(functions) + len(items))

	if data.Total == 0 {
		message := "未匹配到资源。已知完整路径时传 full_code_path；需要找函数 schema 时用 resource_type=function；关键词短语不要按空格拆词，多个候选词用 | 分隔。"
		if displayKeyword == "" {
			message = "未传 keyword 或 full_code_path，当前没有可展示的搜索结果。可以传关键词、完整路径，或 resource_type=tool 查看内置工具。"
		}
		return toolResultWithData(formatSearchMeta(data)+"\n\n"+message, false, data)
	}

	return toolResultWithData(formatSearchOutput(data, requestOutput), false, data)
}

func shouldSearchBuiltinTools(resourceType string, keyword string) bool {
	return resourceType == "tool" || (resourceType == "all" && strings.TrimSpace(keyword) != "")
}

func shouldSearchFunctions(resourceType string, templateType string, capability string, fullCodePath string) bool {
	if templateType != "" || capability != "" {
		return true
	}
	return resourceType == "all" || resourceType == "function" || looksLikeFunctionPath(fullCodePath)
}

func shouldSearchResources(resourceType string) bool {
	return resourceType == "all" || resourceType == "directory" || resourceType == "docs"
}

func looksLikeFunctionPath(fullCodePath string) bool {
	for _, suffix := range []string{".form", ".table", ".chart"} {
		if strings.HasSuffix(fullCodePath, suffix) {
			return true
		}
	}
	return false
}

func searchBuiltinTools(ctx context.Context, registry *ToolRegistry, keyword string, page int, pageSize int) []dto.ToolDef {
	if registry == nil {
		return nil
	}
	allTools, _ := registry.ListTools(ctx, nil)
	keywords := splitSearchKeywords(keyword)
	matched := make([]dto.ToolDef, 0, len(allTools))
	if len(keywords) == 0 {
		matched = append(matched, allTools...)
		return paginateSearchToolDefs(matched, page, pageSize)
	}
	lowerKeywords := make([]string, len(keywords))
	for i, k := range keywords {
		lowerKeywords[i] = strings.ToLower(k)
	}
	for _, tool := range allTools {
		text := strings.ToLower(tool.Name + " " + tool.Description)
		for _, k := range lowerKeywords {
			if strings.Contains(text, k) {
				matched = append(matched, tool)
				break
			}
		}
	}
	return matched
}

func paginateSearchToolDefs(tools []dto.ToolDef, page int, pageSize int) []dto.ToolDef {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || len(tools) == 0 {
		return tools
	}
	start := (page - 1) * pageSize
	if start >= len(tools) {
		return nil
	}
	end := start + pageSize
	if end > len(tools) {
		end = len(tools)
	}
	return tools[start:end]
}

func filterSearchResourceItemsByType(items []*dto.ResourceSearchResult, resourceType string, include bool) []*dto.ResourceSearchResult {
	if resourceType == "" {
		return items
	}
	out := make([]*dto.ResourceSearchResult, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		matches := item.Type == resourceType
		if matches == include {
			out = append(out, item)
		}
	}
	return out
}

func filterSearchFunctionsByFullCodePath(functions []*dto.FunctionSearchResult, fullCodePath string) []*dto.FunctionSearchResult {
	fullCodePath = normalizeWorkspacePath(fullCodePath)
	if fullCodePath == "" {
		return functions
	}
	out := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		if fn == nil {
			continue
		}
		if workspacePathHasPrefix(fn.FullCodePath, fullCodePath) {
			out = append(out, fn)
		}
	}
	return out
}

func paginateSearchFunctions(functions []*dto.FunctionSearchResult, page int, pageSize int) []*dto.FunctionSearchResult {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || len(functions) == 0 {
		return functions
	}
	start := (page - 1) * pageSize
	if start >= len(functions) {
		return nil
	}
	end := start + pageSize
	if end > len(functions) {
		end = len(functions)
	}
	return functions[start:end]
}

func normalizeSearchResourceType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return "all"
	case "tool", "tools", "builtin", "built_in", "built-in":
		return "tool"
	case "function", "functions":
		return "function"
	case "directory", "directories", "dir", "folder", "package":
		return "directory"
	case "docs", "doc", "document":
		return "docs"
	default:
		return "all"
	}
}

func apiSearchResourceType(resourceType string) string {
	switch resourceType {
	case "directory":
		return "package"
	case "docs":
		return "docs"
	default:
		return "all"
	}
}

func normalizeSearchCapability(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "read", "read-only", "create", "update", "delete", "submit", "query":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func filterSearchFunctionsByCapability(functions []*dto.FunctionSearchResult, capability string) []*dto.FunctionSearchResult {
	if capability == "" {
		return functions
	}
	out := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		if searchFunctionHasCapability(fn, capability) {
			out = append(out, fn)
		}
	}
	return out
}

func searchFunctionHasCapability(fn *dto.FunctionSearchResult, capability string) bool {
	if fn == nil {
		return false
	}
	switch capability {
	case "submit":
		return fn.TemplateType == functionschema.TypeForm
	case "query":
		return fn.TemplateType == functionschema.TypeChart
	case "read":
		return fn.TemplateType == functionschema.TypeTable
	case "read-only":
		return fn.TemplateType == functionschema.TypeTable &&
			!hasSearchCallback(fn.Callbacks, "OnTableAddRow") &&
			!hasSearchCallback(fn.Callbacks, "OnTableUpdateRow") &&
			!hasSearchCallback(fn.Callbacks, "OnTableDeleteRows")
	case "create":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableAddRow")
	case "update":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableUpdateRow")
	case "delete":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableDeleteRows")
	default:
		return true
	}
}

func formatSearchMeta(data searchResultData) string {
	parts := []string{
		fmt.Sprintf("page=%d", data.Page),
		fmt.Sprintf("page_size=%d", data.PageSize),
	}
	if data.Keyword != "" {
		parts = append(parts, "keyword="+data.Keyword)
	}
	if data.FullCodePath != "" {
		parts = append(parts, "full_code_path="+data.FullCodePath)
	}
	if data.ResourceType != "" && data.ResourceType != "all" {
		parts = append(parts, "resource_type="+data.ResourceType)
	}
	if data.TemplateType != "" {
		parts = append(parts, "template_type="+data.TemplateType)
	}
	if data.Capability != "" {
		parts = append(parts, "capability="+data.Capability)
	}
	parts = append(parts, fmt.Sprintf("total=%d", data.Total))
	return "搜索参数：" + strings.Join(parts, " | ")
}

func formatSearchOutput(data searchResultData, requestOutput searchRequestOutput) string {
	var buf strings.Builder
	if data.Keyword == "" {
		buf.WriteString("搜索结果：未传 keyword，按完整路径或默认排序返回当前账号可见资源。\n")
	} else {
		buf.WriteString(fmt.Sprintf("搜索结果：%s\n", data.Keyword))
	}
	buf.WriteString(formatSearchMeta(data))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个内置工具\n", len(data.MatchedTools)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个可执行函数\n", len(data.Functions)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个目录/文档资源\n\n", len(data.Items)))

	if len(data.MatchedTools) > 0 {
		buf.WriteString("【内置工具】\n")
		for _, tool := range data.MatchedTools {
			buf.WriteString("- ")
			buf.WriteString(tool.Name)
			buf.WriteString(": ")
			buf.WriteString(searchToolShortDescription(tool.Description))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	if len(data.Functions) > 0 {
		buf.WriteString("【可执行函数】\n")
		buf.WriteString("调用方式：form → run_form_submit，table → 默认先 run_table_search，仅在能力摘要明确支持写入时再用 run_table_create/run_table_update/run_table_delete，chart → run_chart_query。\n")
		for i, fn := range data.Functions {
			buf.WriteString(formatSearchFunctionSummary(i, fn))
		}
		buf.WriteString("\n")
	}

	if len(data.Items) > 0 {
		buf.WriteString("【目录/文档资源】\n")
		for i, item := range data.Items {
			buf.WriteString(formatSearchResourceSummary(i, item))
		}
	}

	if requestOutput == searchRequestOutputBoth && len(data.Functions) > 0 {
		buf.WriteString("\n【函数 Schema JSON】\n")
		buf.WriteString(formatSearchFunctionSchemaJSON(data.Functions))
	}

	return strings.TrimSpace(buf.String())
}

func formatSearchFunctionSchemaJSON(functions []*dto.FunctionSearchResult) string {
	var buf strings.Builder
	for i, fn := range functions {
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, fn.Name))
		buf.WriteString("   full_code_path: ")
		buf.WriteString(fn.FullCodePath)
		buf.WriteString("\n")
		if fn.RunCount > 0 {
			buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", fn.RunCount))
		}
		if fn.Description != "" {
			buf.WriteString("   description: ")
			buf.WriteString(fn.Description)
			buf.WriteString("\n")
		}
		if fn.TemplateType != "" {
			buf.WriteString("   type: ")
			buf.WriteString(fn.TemplateType)
			buf.WriteString("\n")
		}
		if caps := formatSearchFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
			buf.WriteString("   capabilities: ")
			buf.WriteString(caps)
			buf.WriteString("\n")
		}
		if fn.Schema != nil {
			if schemaJSON, err := json.MarshalIndent(fn.Schema, "   ", "  "); err == nil {
				buf.WriteString("   schema: ")
				buf.Write(schemaJSON)
				buf.WriteString("\n")
			}
		}
	}
	return buf.String()
}

func formatSearchFunctionSummary(index int, fn *dto.FunctionSearchResult) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d. %s\n", index+1, fn.Name))
	buf.WriteString("   full_code_path: ")
	buf.WriteString(fn.FullCodePath)
	buf.WriteString("\n")
	if fn.TemplateType != "" {
		buf.WriteString("   type: ")
		buf.WriteString(fn.TemplateType)
		buf.WriteString("\n")
	}
	if caps := formatSearchFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
		buf.WriteString("   capabilities: ")
		buf.WriteString(caps)
		buf.WriteString("\n")
	}
	if fn.RunCount > 0 {
		buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", fn.RunCount))
	}
	if fn.Description != "" {
		buf.WriteString("   description: ")
		buf.WriteString(fn.Description)
		buf.WriteString("\n")
	}

	summaryLines := summarizeSearchSchema(fn.Schema)
	if len(summaryLines) > 0 {
		buf.WriteString("   字段摘要:\n")
		for _, line := range summaryLines {
			buf.WriteString("   ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func formatSearchResourceSummary(index int, item *dto.ResourceSearchResult) string {
	if item == nil {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d. %s\n", index+1, item.Name))
	buf.WriteString("   type: ")
	buf.WriteString(item.Type)
	if item.TemplateType != "" {
		buf.WriteString(" / ")
		buf.WriteString(item.TemplateType)
	}
	buf.WriteString("\n")
	buf.WriteString("   full_code_path: ")
	buf.WriteString(item.FullCodePath)
	buf.WriteString("\n")
	if item.AppUser != "" || item.AppCode != "" {
		buf.WriteString(fmt.Sprintf("   app: %s/%s\n", item.AppUser, item.AppCode))
	}
	if item.MatchSource != "" {
		buf.WriteString("   match_source: ")
		buf.WriteString(item.MatchSource)
		buf.WriteString("\n")
	}
	if item.Description != "" {
		buf.WriteString("   description: ")
		buf.WriteString(item.Description)
		buf.WriteString("\n")
	}
	if item.Snippet != "" && item.Snippet != item.Description {
		buf.WriteString("   snippet: ")
		buf.WriteString(item.Snippet)
		buf.WriteString("\n")
	}
	if item.RunCount > 0 {
		buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", item.RunCount))
	}
	return buf.String()
}

func formatSearchFunctionCapabilities(templateType string, callbacks []string) string {
	switch templateType {
	case functionschema.TypeTable:
		caps := []string{"read"}
		if hasSearchCallback(callbacks, "OnTableAddRow") {
			caps = append(caps, "create")
		}
		if hasSearchCallback(callbacks, "OnTableUpdateRow") {
			caps = append(caps, "update")
		}
		if hasSearchCallback(callbacks, "OnTableDeleteRows") {
			caps = append(caps, "delete")
		}
		if len(caps) == 1 {
			return "read-only"
		}
		return strings.Join(caps, ", ")
	case functionschema.TypeForm:
		return "submit"
	case functionschema.TypeChart:
		return "query"
	default:
		return ""
	}
}

func hasSearchCallback(callbacks []string, target string) bool {
	for _, callback := range callbacks {
		if callback == target {
			return true
		}
	}
	return false
}

func summarizeSearchSchema(schema *functionschema.FunctionSchema) []string {
	if schema == nil {
		return nil
	}
	switch schema.Type {
	case functionschema.TypeForm:
		if schema.Form == nil {
			return nil
		}
		return summarizeSearchFields("输入字段", schema.Form.Request)
	case functionschema.TypeTable:
		raw, _ := functionschema.Marshal(schema)
		lines := summarizeSearchFields("搜索字段", functionschema.TableSearchFields(raw))
		lines = append(lines, summarizeSearchFields("列表字段", functionschema.TableListFields(raw))...)
		lines = append(lines, summarizeSearchFields("新增字段", functionschema.TableCreateFields(raw))...)
		lines = append(lines, summarizeSearchFields("编辑字段", functionschema.TableUpdateFields(raw))...)
		return lines
	case functionschema.TypeChart:
		if schema.Chart == nil {
			return nil
		}
		return summarizeSearchFields("查询字段", schema.Chart.Request)
	default:
		return nil
	}
}

func summarizeSearchFields(label string, fields []*widget.Field) []string {
	if len(fields) == 0 {
		return nil
	}
	lines := make([]string, 0, len(fields)*2+1)
	lines = append(lines, label+"：")
	for _, field := range fields {
		for _, line := range field.LLMSummaryLines(widget.SummaryOptions{
			Mode:     widget.SummaryCompact,
			MaxDepth: 1,
		}) {
			lines = append(lines, "  "+line)
		}
	}
	return lines
}

func searchToolShortDescription(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}
	for _, sep := range []string{"。参数：", "。full_code_path", "\n", "。"} {
		if idx := strings.Index(description, sep); idx > 0 {
			return strings.TrimSpace(description[:idx])
		}
	}
	return description
}
