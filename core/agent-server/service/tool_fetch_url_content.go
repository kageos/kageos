package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/websearch"
)

const maxFetchURLContentCount = 5 // 一次最多拉取多少个链接

type FetchURLContentTool struct{}

type fetchURLContentArgs struct {
	URL      string   `json:"url" schema_desc:"单个网页 URL"`
	URLs     []string `json:"urls" schema_desc:"多个网页 URL，最多 5 个"`
	MaxChars *int     `json:"max_chars" schema_desc:"每条正文最大返回字数"`
}

var fetchURLContentToolDef = toolDefinition[fetchURLContentArgs](
	"fetch_url_content",
	"根据指定 URL（或多个 URL）访问并拉取可读正文。支持 HTML 页面（解析 DOM 取文）、纯文本、Markdown、JSON/XML 等文本类响应；非文本（如二进制）也会返回简短说明（含 Content-Type 与大小）。支持传 url（单个）或 urls（多个，最多 5 个）。",
)

func (t *FetchURLContentTool) Definition() dto.ToolDef {
	return fetchURLContentToolDef
}

func (t *FetchURLContentTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[fetchURLContentArgs](call.Args)
	if err != nil {
		return toolResult("fetch_url_content 参数解析失败: "+err.Error(), true)
	}
	content, isError := runFetchURLContentTool(ctx, args)
	return toolResult(content, isError)
}

// runFetchURLContentTool 按 URL（或 urls 数组）拉取页面正文，支持多链接
func runFetchURLContentTool(ctx context.Context, args fetchURLContentArgs) (string, bool) {
	var urlList []string
	if len(args.URLs) > 0 {
		for _, v := range args.URLs {
			s := strings.TrimSpace(v)
			if s != "" {
				urlList = append(urlList, s)
			}
		}
		if len(urlList) > maxFetchURLContentCount {
			urlList = urlList[:maxFetchURLContentCount]
		}
	}
	if len(urlList) == 0 {
		rawURL := strings.TrimSpace(args.URL)
		if rawURL == "" {
			return "fetch_url_content 需填 url（单个）或 urls（多个）。", true
		}
		urlList = []string{rawURL}
	}
	for i, u := range urlList {
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			urlList[i] = "https://" + u
		}
	}

	maxChars := 3000
	if args.MaxChars != nil && *args.MaxChars > 0 {
		maxChars = *args.MaxChars
		if maxChars > 20000 {
			maxChars = 20000
		}
	}

	var b strings.Builder
	for i, rawURL := range urlList {
		title, body := websearch.FetchURLContent(ctx, rawURL, maxChars)
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("【第 %d 个链接】%s\n", i+1, rawURL))
		if title != "" {
			b.WriteString("标题: " + title + "\n\n")
		}
		if body != "" {
			b.WriteString("正文: " + body)
		} else {
			b.WriteString("（请求失败、无响应体或无法建立连接）")
		}
	}
	if b.Len() == 0 {
		return "所有链接均无法拉取内容。", false
	}
	return b.String(), false
}
