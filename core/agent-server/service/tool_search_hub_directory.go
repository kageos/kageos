package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type SearchHubDirectoryTool struct{}

type searchHubDirectoryArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"目录完整路径，用于查询单个应用"`
	Search       string `json:"search" schema_desc:"搜索关键词"`
	Page         *int   `json:"page" schema_desc:"页码"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数"`
}

var searchHubDirectoryToolDef = toolDefinition[searchHubDirectoryArgs](
	"search_hub_directory",
	"在应用中心（Hub）搜索应用，或按路径查询单个目录在 Hub 上的信息。① 按关键词搜索：传 search（可选，不传或传空则返回全部应用）；支持多关键字「或」搜索，用 | 分隔，例如：美发|理发|美容|预约，表示匹配其中任意一词即可；可传 page、page_size（可选）。② 按路径查当前目录在 Hub 上的信息：传 full_code_path（如 /user/app/plugins/xxx），可查看该路径是否已上架、copy_url、star_count 等。返回含 copy_url（用于 copy_directory）、star_count、download_count 等。",
)

func (t *SearchHubDirectoryTool) Definition() dto.ToolDef {
	return searchHubDirectoryToolDef
}

func (t *SearchHubDirectoryTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchHubDirectoryArgs](call.Args)
	if err != nil {
		return toolResult("search_hub_directory 参数解析失败: "+err.Error(), true)
	}
	content, isError := runSearchHubDirectoryTool(ctx, args)
	return toolResult(content, isError)
}

func runSearchHubDirectoryTool(ctx context.Context, args searchHubDirectoryArgs) (string, bool) {
	fullCodePath := normalizeAbsoluteToolPath(args.FullCodePath)
	if fullCodePath != "" {
		detail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
			FullCodePath: fullCodePath,
			IncludeTree:  false,
		})
		if err != nil {
			return fmt.Sprintf("该路径在应用中心未找到或未上架：%s。可先用 publish_to_hub 发布后再查询。", fullCodePath), false
		}
		if detail == nil {
			return fmt.Sprintf("路径 %s 在应用中心暂无信息（可能未上架）。", fullCodePath), false
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("应用中心 - 路径 %s 的信息：\n\n", detail.FullCodePath))
		b.WriteString(fmt.Sprintf("名称: %s\n", detail.Name))
		if detail.Description != "" {
			desc := detail.Description
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}
			b.WriteString("描述: " + desc + "\n")
		}
		b.WriteString(fmt.Sprintf("路径: %s | 发布者: %s | 版本: %s\n", detail.FullCodePath, detail.PublisherUsername, detail.Version))
		b.WriteString(fmt.Sprintf("星 %d | 克隆 %d 次 | 目录 %d / 文件 %d / 函数 %d\n", detail.StarCount, detail.DownloadCount, detail.DirectoryCount, detail.FileCount, detail.FunctionCount))
		if detail.CopyURL != "" {
			b.WriteString("复制链接（用于 copy_directory）: " + detail.CopyURL + "\n")
			b.WriteString("复制时 target_directory 填当前工作区路径（目标父目录），不要填「父路径/子目录名」。\n")
		}
		return b.String(), false
	}

	req := &dto.GetHubDirectoryListReq{
		Page:     1,
		PageSize: 10,
	}
	if s := strings.TrimSpace(args.Search); s != "" {
		req.Search = s
	}
	if args.Page != nil && *args.Page > 0 {
		req.Page = *args.Page
	}
	if args.PageSize != nil && *args.PageSize > 0 {
		req.PageSize = *args.PageSize
		if req.PageSize > 50 {
			req.PageSize = 50
		}
	}
	resp, err := apicall.GetHubDirectoryList(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[SearchHub] GetHubDirectoryList 失败: %v", err)
		return "search_hub_directory 调用失败: " + err.Error(), true
	}
	if len(resp.Items) == 0 {
		return fmt.Sprintf("应用中心共 %d 条结果，当前页无数据。可调整 search（支持多关键字用 | 分隔）或 page 再试。", resp.Total), false
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("应用中心搜索结果（共 %d 条，当前第 %d 页）：\n\n", resp.Total, resp.Page))
	for i, item := range resp.Items {
		b.WriteString(fmt.Sprintf("【%d】%s\n", i+1, item.Name))
		if item.Description != "" {
			desc := item.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			b.WriteString("  描述: " + desc + "\n")
		}
		b.WriteString(fmt.Sprintf("  路径: %s | 发布者: %s | 星 %d | 克隆 %d 次\n", item.FullCodePath, item.PublisherUsername, item.StarCount, item.DownloadCount))
		if item.CopyURL != "" {
			b.WriteString("  复制链接（用于 copy_directory）: " + item.CopyURL + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("使用 copy_directory(source_directory=\"上面的复制链接\", target_directory=\"/你的用户/你的应用/当前目录\") 即可将应用复制到本地；target_directory 填当前工作区路径（目标父目录），会在其下自动创建与源同名的子目录，不要填「父目录/子目录名」。")
	return b.String(), false
}
