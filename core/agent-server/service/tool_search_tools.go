package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type SearchToolsTool struct{ registry *ToolRegistry }

type searchToolsArgs struct {
	Keyword      string `json:"keyword" schema_desc:"搜索关键词，支持竖线分隔多个关键词"`
	TemplateType string `json:"template_type" schema_desc:"函数类型过滤" schema_enum:"form,table,chart"`
	Limit        *int   `json:"limit" schema_desc:"最多返回条数"`
}

var searchToolsToolDef = toolDefinition[searchToolsArgs](
	"search_tools",
	"按关键词搜索可用工具：返回「内置工具」与「system 用户下已注册的表单/表格/图表函数」。keyword 可选：不传则按调用次数返回高频已注册函数；传则按关键词匹配。多关键词用竖线 | 分隔（OR 语义），如 折线图|chart|画图。template_type 建议杂活传 form。",
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

// runSearchToolsTool 按关键词搜索可用工具（内置工具 + system 用户下已注册 Form/Table/Chart）
func runSearchToolsTool(ctx context.Context, registry *ToolRegistry, args searchToolsArgs) (string, bool) {
	keywordRaw := strings.TrimSpace(args.Keyword)
	keywords := splitSearchKeywords(keywordRaw)
	templateType := strings.TrimSpace(args.TemplateType)
	limit := 20
	if args.Limit != nil && *args.Limit > 0 {
		limit = *args.Limit
		if limit > 50 {
			limit = 50
		}
	}
	var buf strings.Builder

	if len(keywords) > 0 && registry != nil {
		allTools, _ := registry.ListTools(ctx, nil)
		lowerKeywords := make([]string, len(keywords))
		for i, k := range keywords {
			lowerKeywords[i] = strings.ToLower(k)
		}
		var matchedTools []dto.ToolDef
		for _, t := range allTools {
			text := strings.ToLower(t.Name + " " + t.Description)
			for _, k := range lowerKeywords {
				if strings.Contains(text, k) {
					matchedTools = append(matchedTools, t)
					break
				}
			}
		}
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
	if len(functions) > 0 {
		if keywordRaw == "" {
			buf.WriteString("【已注册函数】（按调用次数从高到低，仅 system 用户下）\n")
		} else {
			buf.WriteString("【已注册函数】（仅 system 用户下）调用方式：form → run_form_submit，table → run_table_search/run_table_create/run_table_update，chart → run_chart_query。\n")
		}
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
			if len(fn.Request) > 0 {
				if reqJSON, err := json.MarshalIndent(fn.Request, "   ", "  "); err == nil {
					buf.WriteString("   request: ")
					buf.Write(reqJSON)
					buf.WriteString("\n")
				}
			}
		}
	} else if buf.Len() == 0 {
		if keywordRaw == "" {
			buf.WriteString("当前 system 用户下暂无已注册函数；可传 keyword 按关键词搜索，或使用 search_hub_directory 搜应用市场。")
		} else {
			buf.WriteString("未匹配到任何可用工具（内置工具或 system 用户下已注册函数），可考虑 search_hub_directory 搜应用市场，或创建新目录并按「创建项目」流程（先 PRD、用户确认后再写代码）。")
		}
	}
	return buf.String(), false
}
