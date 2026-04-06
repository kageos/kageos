package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type SearchReplaceFileTool struct{}

type searchReplaceItemArgs struct {
	SearchString  string `json:"search_string" schema_desc:"要被替换的原文" schema_required:"true"`
	ReplaceString string `json:"replace_string" schema_desc:"替换后的内容"`
	ExpectedCount int    `json:"expected_count" schema_desc:"预期匹配次数"`
}

type searchReplaceFileArgs struct {
	Directory         string                  `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath      string                  `json:"full_code_path" schema_ignore:"true"`
	FileName          string                  `json:"file_name" schema_desc:"目标文件名" schema_required:"true"`
	Replacements      []searchReplaceItemArgs `json:"replacements" schema_desc:"替换列表" schema_required:"true"`
	AllOrNothing      *bool                   `json:"all_or_nothing" schema_desc:"是否全部匹配成功才落盘"`
	ReturnFullContent *bool                   `json:"return_full_content" schema_desc:"是否返回替换后的完整文件内容"`
}

var searchReplaceFileToolDef = toolDefinition[searchReplaceFileArgs](
	"search_replace_file",
	"在指定目录下的 .go 文件中做「查找并替换」：只改匹配到的片段，不重写整文件。必填：directory（或当前目录）、file_name、replacements（替换列表，每项含 search_string、replace_string、expected_count 可选默认 1）。all_or_nothing 默认 true：仅当所有项的实际匹配次数等于 expected_count 时才落盘，否则不写入。search_string 必须与文件内容完全一致（含空格、制表符、换行）；使用前建议先用 read_go_file 读取后从实际内容复制。示例：replacements: [{ \"search_string\": \"原文\", \"replace_string\": \"新文\", \"expected_count\": 1 }]。仅修改代码、不编译工作空间；若需生效改完后需调用 build_workspace。",
)

func (t *SearchReplaceFileTool) Definition() dto.ToolDef {
	return searchReplaceFileToolDef
}

func (t *SearchReplaceFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[searchReplaceFileArgs](call.Args)
	if err != nil {
		return toolResult("search_replace_file 参数解析失败: "+err.Error(), true)
	}
	content, isError := runSearchReplaceFileTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// parseReplacementsFromArgs 从工具参数中解析 replacements 数组为 []dto.ReplaceItem
func parseReplacementsFromArgs(items []searchReplaceItemArgs) ([]dto.ReplaceItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	out := make([]dto.ReplaceItem, 0, len(items))
	for _, item := range items {
		search := strings.TrimSpace(item.SearchString)
		if search == "" {
			return nil, false
		}
		out = append(out, dto.ReplaceItem{
			SearchString:  search,
			ReplaceString: item.ReplaceString,
			ExpectedCount: item.ExpectedCount,
		})
	}
	return out, true
}

// runSearchReplaceFileTool 文件 search-replace（统一批量：多组替换同一文件，全部生效才落盘）
func runSearchReplaceFileTool(ctx context.Context, args searchReplaceFileArgs, currentFullCodePath string) (string, bool) {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return "search_replace_file 缺少参数 file_name。", true
	}
	replacements, ok := parseReplacementsFromArgs(args.Replacements)
	if !ok || len(replacements) == 0 {
		return "search_replace_file 缺少参数 replacements（替换列表，每项含 search_string、replace_string、expected_count 可选）。", true
	}
	allOrNothing := true
	if args.AllOrNothing != nil {
		allOrNothing = *args.AllOrNothing
	}
	returnFullContent := false
	if args.ReturnFullContent != nil {
		returnFullContent = *args.ReturnFullContent
	}
	req := &dto.ReplaceFileContentReq{
		FullCodePath:      targetPath,
		FileName:          fileName,
		Replacements:      replacements,
		AllOrNothing:      allOrNothing,
		ReturnFullContent: returnFullContent,
	}
	resp, err := apicall.ReplaceFileContent(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[SearchReplaceFile] ReplaceFileContent 失败: %v", err)
		return "search_replace_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		msg := "search_replace_file: " + resp.Message
		if len(resp.Details) > 0 {
			for _, d := range resp.Details {
				msg += fmt.Sprintf("\n第 %d 项预期匹配 %d 次，实际匹配 %d 次；请将该项 expected_count 改为 %d 或核对内容后再调用。", d.Index+1, d.ExpectedCount, d.ActualCount, d.ActualCount)
			}
		}
		return msg, true
	}
	msg := fmt.Sprintf("已替换: 目录=%s, 文件=%s, 共 %d 处。修改已落盘，但未编译工作空间；若需生效请调用 build_workspace 更新工作空间。", targetPath, fileName, resp.ReplaceCount)
	if resp.FullContent != "" {
		msg += "\n\n替换后完整内容：\n```go\n" + resp.FullContent + "\n```"
	}
	return msg, false
}
