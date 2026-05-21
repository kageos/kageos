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

type SearchToolsTool struct{ registry *ToolRegistry }

type searchToolsArgs struct {
	Keyword      string `json:"keyword" schema_desc:"搜索关键词，支持竖线分隔多个关键词"`
	TemplateType string `json:"template_type" schema_desc:"函数类型过滤" schema_enum:"form,table,chart"`
	Scope        string `json:"scope" schema_desc:"搜索范围：system=官方/system 函数（默认，兼容旧行为），visible=当前用户可见函数，current_user=当前用户下函数，current_app=当前工作区应用内函数" schema_enum:"system,visible,current_user,current_app"`
	User         string `json:"user" schema_desc:"按用户名过滤；传入后优先于 scope 推导"`
	App          string `json:"app" schema_desc:"按应用 code 过滤；通常和 user 搭配使用"`
	Capability   string `json:"capability" schema_desc:"函数能力过滤：form 支持 submit；chart 支持 query；table 支持 read/create/update/delete/read-only" schema_enum:"read,read-only,create,update,delete,submit,query"`
	Page         *int   `json:"page" schema_desc:"页码，默认 1"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数，默认使用 limit 或 20，最多 100"`
	Limit        *int   `json:"limit" schema_desc:"最多返回条数"`
	SchemaOutput string `json:"schema_output" schema_desc:"schema 输出方式：summary=字段摘要（默认），both=字段摘要和原始 JSON" schema_enum:"summary,both"`
}

var searchToolsToolDef = toolDefinition[searchToolsArgs](
	"search_tools",
	"按关键词搜索可执行工具：返回「内置工具」与已注册的表单/表格/图表函数。默认 scope=system 搜官方/system 函数；可用 scope=visible 搜当前用户可见函数，scope=current_app 搜当前工作区应用，或传 user/app 精确过滤。keyword 可选：不传则按调用次数返回高频函数；多关键词用竖线 | 分隔（OR 语义）。可用 template_type 和 capability 缩小到 form/table/chart 或 read/create/update/delete/submit/query 等能力。执行表单/表格/图表前用 schema_output=summary 或 both 获取字段摘要，确认字段名、必填项、枚举值和文件字段后再调用执行工具。",
)

func (t *SearchToolsTool) Definition() dto.ToolDef {
	return searchToolsToolDef
}

func (t *SearchToolsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchToolsArgs](call.Args)
	if err != nil {
		return toolResult("search_tools 参数解析失败: "+err.Error(), true)
	}
	return runSearchToolsTool(ctx, t.registry, args, call.FullCodePath)
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

type searchToolsRequestOutput string

const (
	searchToolsRequestOutputSummary searchToolsRequestOutput = "summary"
	searchToolsRequestOutputBoth    searchToolsRequestOutput = "both"
)

func normalizeSearchToolsRequestOutput(raw string) searchToolsRequestOutput {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(searchToolsRequestOutputSummary):
		return searchToolsRequestOutputSummary
	case string(searchToolsRequestOutputBoth):
		return searchToolsRequestOutputBoth
	default:
		return searchToolsRequestOutputSummary
	}
}

type searchToolsResultData struct {
	Keyword      string                      `json:"keyword"`
	Scope        string                      `json:"scope"`
	User         string                      `json:"user,omitempty"`
	App          string                      `json:"app,omitempty"`
	TemplateType string                      `json:"template_type,omitempty"`
	Capability   string                      `json:"capability,omitempty"`
	Page         int                         `json:"page"`
	PageSize     int                         `json:"page_size"`
	Total        int64                       `json:"total"`
	MatchedTools []dto.ToolDef               `json:"matched_tools,omitempty"`
	Functions    []*dto.FunctionSearchResult `json:"functions,omitempty"`
}

func normalizeSearchToolsPageSize(args searchToolsArgs) int {
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

func normalizeSearchToolsPage(args searchToolsArgs) int {
	if args.Page != nil && *args.Page > 0 {
		return *args.Page
	}
	return 1
}

// runSearchToolsTool 按关键词搜索可用工具（内置工具 + 已注册 Form/Table/Chart）
func runSearchToolsTool(ctx context.Context, registry *ToolRegistry, args searchToolsArgs, currentFullCodePath string) ToolResult {
	keywordRaw := strings.TrimSpace(args.Keyword)
	keywords := splitSearchKeywords(keywordRaw)
	templateType := strings.TrimSpace(args.TemplateType)
	capability := normalizeSearchToolsCapability(args.Capability)
	requestOutput := normalizeSearchToolsRequestOutput(args.SchemaOutput)
	page := normalizeSearchToolsPage(args)
	pageSize := normalizeSearchToolsPageSize(args)
	fetchPage := page
	fetchPageSize := pageSize
	if capability != "" {
		fetchPage = 1
		fetchPageSize = 100
	}
	user, app, scope := resolveSearchScopeUserApp(args.Scope, args.User, args.App, currentFullCodePath, searchScopeSystem)

	matchedTools := make([]dto.ToolDef, 0)
	if len(keywords) > 0 && registry != nil {
		allTools, _ := registry.ListTools(ctx, nil)
		lowerKeywords := make([]string, len(keywords))
		for i, k := range keywords {
			lowerKeywords[i] = strings.ToLower(k)
		}
		for _, t := range allTools {
			text := strings.ToLower(t.Name + " " + t.Description)
			for _, k := range lowerKeywords {
				if strings.Contains(text, k) {
					matchedTools = append(matchedTools, t)
					break
				}
			}
		}
	}

	resp, err := apicall.SearchFunctions(ctx, &dto.SearchFunctionsReq{
		User:         user,
		App:          app,
		Keyword:      keywordRaw,
		TemplateType: templateType,
		Page:         fetchPage,
		PageSize:     fetchPageSize,
	})
	functions := make([]*dto.FunctionSearchResult, 0)
	var total int64
	if err != nil {
		logger.Warnf(ctx, "[SearchTools] SearchFunctions err: %v", err)
	} else if resp != nil {
		functions = resp.Functions
		total = resp.Total
	}
	if capability != "" {
		functions = filterSearchToolFunctionsByCapability(functions, capability)
		total = int64(len(functions))
		functions = paginateSearchToolFunctions(functions, page, pageSize)
	}

	data := searchToolsResultData{
		Keyword:      keywordRaw,
		Scope:        scope,
		User:         user,
		App:          app,
		TemplateType: templateType,
		Capability:   capability,
		Page:         page,
		PageSize:     pageSize,
		Total:        total,
		MatchedTools: matchedTools,
		Functions:    functions,
	}

	if len(functions) == 0 && len(matchedTools) == 0 {
		if keywordRaw == "" {
			return toolResultWithData(formatSearchToolsNoResultMessage(data, "当前搜索范围下暂无已注册函数；可传 keyword 按关键词搜索。"), false, data)
		}
		return toolResultWithData(formatSearchToolsNoResultMessage(data, "未匹配到任何可用工具（内置工具或已注册函数）。如果用户要新建长期应用，先 change_role 到产品经理（product_manager），输出 PRD 并等用户确认后再进入应用开发工程师（app_developer）写代码。"), false, data)
	}
	content := formatSearchToolsOutput(keywordRaw, matchedTools, functions, requestOutput)
	content = formatSearchToolsSearchMeta(data) + "\n\n" + content
	return toolResultWithData(content, false, data)
}

func formatSearchToolsOutput(keywordRaw string, matchedTools []dto.ToolDef, functions []*dto.FunctionSearchResult, requestOutput searchToolsRequestOutput) string {
	var buf strings.Builder
	if keywordRaw == "" {
		buf.WriteString("搜索结果：未传 keyword，返回 system 用户下的高频已注册函数。\n")
	} else {
		buf.WriteString(fmt.Sprintf("搜索结果：关键词「%s」\n", keywordRaw))
	}
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个内置工具\n", len(matchedTools)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个已注册函数\n\n", len(functions)))

	if len(matchedTools) > 0 {
		buf.WriteString("【内置工具】\n")
		for _, t := range matchedTools {
			buf.WriteString("- ")
			buf.WriteString(t.Name)
			buf.WriteString(": ")
			buf.WriteString(searchToolShortDescription(t.Description))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	if len(functions) > 0 {
		buf.WriteString("【已注册函数摘要】\n")
		if keywordRaw == "" {
			buf.WriteString("按调用次数从高到低，仅 system 用户下。\n")
		} else {
			buf.WriteString("调用方式：form → run_form_submit，table → 默认先 run_table_search，仅在函数能力摘要明确支持写入时再用 run_table_create/run_table_update/run_table_delete，chart → run_chart_query。\n")
		}
		for i, fn := range functions {
			buf.WriteString(formatSearchToolFunctionSummary(i, fn))
		}
	}

	if requestOutput == searchToolsRequestOutputBoth {
		buf.WriteString("\n【已注册函数 Schema JSON】\n")
		buf.WriteString(formatSearchToolsFunctionSchemaJSON(functions))
	}

	return strings.TrimSpace(buf.String())
}

func normalizeSearchToolsCapability(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "read", "read-only", "create", "update", "delete", "submit", "query":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func filterSearchToolFunctionsByCapability(functions []*dto.FunctionSearchResult, capability string) []*dto.FunctionSearchResult {
	if capability == "" {
		return functions
	}
	out := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		if searchToolFunctionHasCapability(fn, capability) {
			out = append(out, fn)
		}
	}
	return out
}

func paginateSearchToolFunctions(functions []*dto.FunctionSearchResult, page int, pageSize int) []*dto.FunctionSearchResult {
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

func searchToolFunctionHasCapability(fn *dto.FunctionSearchResult, capability string) bool {
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
			!hasSearchToolCallback(fn.Callbacks, "OnTableAddRow") &&
			!hasSearchToolCallback(fn.Callbacks, "OnTableUpdateRow") &&
			!hasSearchToolCallback(fn.Callbacks, "OnTableDeleteRows")
	case "create":
		return fn.TemplateType == functionschema.TypeTable && hasSearchToolCallback(fn.Callbacks, "OnTableAddRow")
	case "update":
		return fn.TemplateType == functionschema.TypeTable && hasSearchToolCallback(fn.Callbacks, "OnTableUpdateRow")
	case "delete":
		return fn.TemplateType == functionschema.TypeTable && hasSearchToolCallback(fn.Callbacks, "OnTableDeleteRows")
	default:
		return true
	}
}

func formatSearchToolsSearchMeta(data searchToolsResultData) string {
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
	if data.TemplateType != "" {
		parts = append(parts, "template_type="+data.TemplateType)
	}
	if data.Capability != "" {
		parts = append(parts, "capability="+data.Capability)
	}
	if data.Total > 0 {
		parts = append(parts, fmt.Sprintf("total=%d", data.Total))
	}
	return "搜索参数：" + strings.Join(parts, " | ")
}

func formatSearchToolsNoResultMessage(data searchToolsResultData, message string) string {
	return formatSearchToolsSearchMeta(data) + "\n\n" + message
}

func formatSearchToolsFunctionSchemaJSON(functions []*dto.FunctionSearchResult) string {
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
		if caps := formatSearchToolFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
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

func formatSearchToolFunctionSummary(index int, fn *dto.FunctionSearchResult) string {
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
	if caps := formatSearchToolFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
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

	summaryLines := summarizeSearchToolSchema(fn.Schema)
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

func formatSearchToolFunctionCapabilities(templateType string, callbacks []string) string {
	switch templateType {
	case functionschema.TypeTable:
		caps := []string{"read"}
		if hasSearchToolCallback(callbacks, "OnTableAddRow") {
			caps = append(caps, "create")
		}
		if hasSearchToolCallback(callbacks, "OnTableUpdateRow") {
			caps = append(caps, "update")
		}
		if hasSearchToolCallback(callbacks, "OnTableDeleteRows") {
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

func hasSearchToolCallback(callbacks []string, target string) bool {
	for _, callback := range callbacks {
		if callback == target {
			return true
		}
	}
	return false
}

func summarizeSearchToolSchema(schema *functionschema.FunctionSchema) []string {
	if schema == nil {
		return nil
	}
	switch schema.Type {
	case functionschema.TypeForm:
		if schema.Form == nil {
			return nil
		}
		return summarizeSearchToolFields("输入字段", schema.Form.Request)
	case functionschema.TypeTable:
		raw, _ := functionschema.Marshal(schema)
		lines := summarizeSearchToolFields("搜索字段", functionschema.TableSearchFields(raw))
		lines = append(lines, summarizeSearchToolFields("列表字段", functionschema.TableListFields(raw))...)
		lines = append(lines, summarizeSearchToolFields("新增字段", functionschema.TableCreateFields(raw))...)
		lines = append(lines, summarizeSearchToolFields("编辑字段", functionschema.TableUpdateFields(raw))...)
		return lines
	case functionschema.TypeChart:
		if schema.Chart == nil {
			return nil
		}
		return summarizeSearchToolFields("查询字段", schema.Chart.Request)
	default:
		return nil
	}
}

func summarizeSearchToolFields(label string, fields []*widget.Field) []string {
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
