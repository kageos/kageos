package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type SearchToolsTool struct{ registry *ToolRegistry }

type searchToolsArgs struct {
	Keyword       string `json:"keyword" schema_desc:"搜索关键词，支持竖线分隔多个关键词"`
	TemplateType  string `json:"template_type" schema_desc:"函数类型过滤" schema_enum:"form,table,chart"`
	Limit         *int   `json:"limit" schema_desc:"最多返回条数"`
	RequestOutput string `json:"request_output" schema_desc:"request 输出方式：summary=字段摘要（默认），json=原始 JSON，both=同时输出" schema_enum:"summary,json,both"`
}

var searchToolsToolDef = toolDefinition[searchToolsArgs](
	"search_tools",
	"按关键词搜索可用工具：返回「内置工具」与「system 用户下已注册的表单/表格/图表函数」。keyword 可选：不传则按调用次数返回高频已注册函数；传则按关键词匹配。多关键词用竖线 | 分隔（OR 语义），如 折线图|chart|画图。template_type 建议杂活传 form。执行表单/表格/图表前可用 request_output=summary 或 both 获取字段摘要，确认字段名、必填项、枚举值和文件字段后再调用执行工具。",
)

func (t *SearchToolsTool) Definition() dto.ToolDef {
	return searchToolsToolDef
}

func (t *SearchToolsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchToolsArgs](call.Args)
	if err != nil {
		return toolResult("search_tools 参数解析失败: "+err.Error(), true)
	}
	content, isError := runSearchToolsTool(ctx, t.registry, args)
	return toolResult(content, isError)
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
	searchToolsRequestOutputJSON    searchToolsRequestOutput = "json"
	searchToolsRequestOutputBoth    searchToolsRequestOutput = "both"
)

func normalizeSearchToolsRequestOutput(raw string) searchToolsRequestOutput {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(searchToolsRequestOutputSummary):
		return searchToolsRequestOutputSummary
	case string(searchToolsRequestOutputJSON):
		return searchToolsRequestOutputJSON
	case string(searchToolsRequestOutputBoth):
		return searchToolsRequestOutputBoth
	default:
		return searchToolsRequestOutputSummary
	}
}

// runSearchToolsTool 按关键词搜索可用工具（内置工具 + system 用户下已注册 Form/Table/Chart）
func runSearchToolsTool(ctx context.Context, registry *ToolRegistry, args searchToolsArgs) (string, bool) {
	keywordRaw := strings.TrimSpace(args.Keyword)
	keywords := splitSearchKeywords(keywordRaw)
	templateType := strings.TrimSpace(args.TemplateType)
	requestOutput := normalizeSearchToolsRequestOutput(args.RequestOutput)
	limit := 20
	if args.Limit != nil && *args.Limit > 0 {
		limit = *args.Limit
		if limit > 50 {
			limit = 50
		}
	}
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
		User:         "system",
		App:          "",
		Keyword:      keywordRaw,
		TemplateType: templateType,
		Page:         1,
		PageSize:     limit,
	})
	functions := make([]*dto.FunctionSearchResult, 0)
	if err != nil {
		logger.Warnf(ctx, "[SearchTools] SearchFunctions err: %v", err)
	} else if resp != nil {
		functions = resp.Functions
	}
	if len(functions) == 0 && len(matchedTools) == 0 {
		if keywordRaw == "" {
			return "当前 system 用户下暂无已注册函数；可传 keyword 按关键词搜索，或使用 search_hub_directory 搜应用市场。", false
		}
		return "未匹配到任何可用工具（内置工具或 system 用户下已注册函数），可考虑 search_hub_directory 搜应用市场，或创建新目录并按「创建项目」流程（先 PRD、用户确认后再写代码）。", false
	}
	return formatSearchToolsOutput(keywordRaw, matchedTools, functions, requestOutput), false
}

func formatSearchToolsOutput(keywordRaw string, matchedTools []dto.ToolDef, functions []*dto.FunctionSearchResult, requestOutput searchToolsRequestOutput) string {
	if requestOutput == searchToolsRequestOutputJSON {
		return formatSearchToolsLegacyOutput(keywordRaw, matchedTools, functions)
	}

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
			buf.WriteString("调用方式：form → run_form_submit，table → 默认先 run_table_search，仅在函数能力摘要明确支持写入时再用 run_table_create/run_table_update，chart → run_chart_query。\n")
		}
		for i, fn := range functions {
			buf.WriteString(formatSearchToolFunctionSummary(i, fn))
		}
	}

	if requestOutput == searchToolsRequestOutputBoth {
		buf.WriteString("\n【已注册函数原始 request JSON】\n")
		buf.WriteString(formatSearchToolsLegacyFunctionRequests(functions))
	}

	return strings.TrimSpace(buf.String())
}

func formatSearchToolsLegacyOutput(keywordRaw string, matchedTools []dto.ToolDef, functions []*dto.FunctionSearchResult) string {
	var buf strings.Builder
	if len(matchedTools) > 0 {
		buf.WriteString("【内置工具】\n")
		for _, t := range matchedTools {
			buf.WriteString("- ")
			buf.WriteString(t.Name)
			buf.WriteString("：")
			buf.WriteString(t.Description)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
	if len(functions) > 0 {
		if keywordRaw == "" {
			buf.WriteString("【已注册函数】（按调用次数从高到低，仅 system 用户下）\n")
		} else {
			buf.WriteString("【已注册函数】（仅 system 用户下）调用方式：form → run_form_submit，table → 默认先 run_table_search，仅在函数能力摘要明确支持写入时再用 run_table_create/run_table_update，chart → run_chart_query。\n")
		}
		buf.WriteString(formatSearchToolsLegacyFunctionRequests(functions))
	}
	return strings.TrimSpace(buf.String())
}

func formatSearchToolsLegacyFunctionRequests(functions []*dto.FunctionSearchResult) string {
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
		if len(fn.Request) > 0 {
			if reqJSON, err := json.MarshalIndent(fn.Request, "   ", "  "); err == nil {
				buf.WriteString("   request: ")
				buf.Write(reqJSON)
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

	summaryLines, err := summarizeSearchToolRequestFields(fn.Request)
	if err != nil {
		buf.WriteString("   字段摘要解析失败，将回退使用原始 request JSON。\n")
		return buf.String()
	}
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

func formatSearchToolFunctionCapabilities(templateType, callbacks string) string {
	switch templateType {
	case "table":
		caps := []string{"read"}
		if hasSearchToolCallback(callbacks, "OnTableAddRow") {
			caps = append(caps, "create")
		}
		if hasSearchToolCallback(callbacks, "OnTableCreateInBatches") {
			caps = append(caps, "batch-create")
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
	case "form":
		return "submit"
	case "chart":
		return "query"
	default:
		return ""
	}
}

func hasSearchToolCallback(callbacks, target string) bool {
	for _, callback := range strings.Split(callbacks, ",") {
		if strings.TrimSpace(callback) == target {
			return true
		}
	}
	return false
}

func summarizeSearchToolRequestFields(raw []interface{}) ([]string, error) {
	fields, err := widget.DecodeFields(raw)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, nil
	}
	lines := make([]string, 0, len(fields)*2)
	for _, field := range fields {
		lines = append(lines, field.LLMSummaryLines(widget.SummaryOptions{
			Mode:     widget.SummaryCompact,
			MaxDepth: 1,
		})...)
	}
	return lines, nil
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
