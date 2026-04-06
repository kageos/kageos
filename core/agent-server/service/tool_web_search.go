package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/websearch"
)

type WebSearchTool struct{}

type webSearchArgs struct {
	Keyword string `json:"keyword" schema_desc:"搜索关键词" schema_required:"true"`
	Limit   *int   `json:"limit" schema_desc:"最多返回条数"`
}

var webSearchToolDef = toolDefinition[webSearchArgs](
	"web_search",
	"在互联网上搜索知识、概念或资料。默认使用百度搜索，必要时回退必应（国内可直接访问，不调用第三方付费 API）。当需要最新信息、概念解释、技术文档或事实查证时调用。",
)

func (t *WebSearchTool) Definition() dto.ToolDef {
	return webSearchToolDef
}

func (t *WebSearchTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[webSearchArgs](call.Args)
	if err != nil {
		return toolResult("web_search 参数解析失败: "+err.Error(), true)
	}
	content, isError := runWebSearchTool(ctx, args)
	return toolResult(content, isError)
}

// runWebSearchTool 调用 pkg/websearch（默认百度、可回退必应），返回格式化文本供模型使用
func runWebSearchTool(ctx context.Context, args webSearchArgs) (string, bool) {
	keyword := strings.TrimSpace(args.Keyword)
	if keyword == "" {
		return "web_search 必填 keyword（搜索关键词）。", true
	}
	limit := 10
	if args.Limit != nil && *args.Limit > 0 {
		limit = *args.Limit
		if limit > 20 {
			limit = 20
		}
	}
	results, err := websearch.Search(ctx, keyword, limit)
	if err != nil {
		logger.Warnf(ctx, "[web_search] Search 失败: %v", err)
		return "web_search 暂时不可用，请稍后再试。", false
	}
	if len(results) == 0 {
		return "未找到与「" + keyword + "」相关的搜索结果。可尝试更换关键词。", false
	}
	const maxSnippetLen = 300
	const maxBodyLen = 1500
	var b strings.Builder
	b.WriteString("【网络搜索结果】关键词：「" + keyword + "」共 " + fmt.Sprintf("%d", len(results)) + " 条\n\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.URL != "" {
			b.WriteString("   链接: " + r.URL + "\n")
		}
		if r.Snippet != "" {
			snippet := r.Snippet
			if len(snippet) > maxSnippetLen {
				snippet = snippet[:maxSnippetLen] + "..."
			}
			b.WriteString("   摘要: " + snippet + "\n")
		}
		if r.Body != "" {
			body := r.Body
			if len(body) > maxBodyLen {
				body = body[:maxBodyLen] + "..."
			}
			b.WriteString("   正文: " + body + "\n")
		}
		b.WriteString("\n")
	}
	return b.String(), false
}
