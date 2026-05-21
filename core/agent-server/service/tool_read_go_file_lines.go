package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

type ReadGoFileLinesTool struct{}

type readGoFileLinesArgs struct {
	Directory    string `json:"directory" schema_desc:"代码目录，不传则当前目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"目标文件名" schema_required:"true"`
	LineRanges   string `json:"line_ranges" schema_desc:"行号范围，如 10-12,20-30"`
}

var readGoFileLinesToolDef = toolDefinition[readGoFileLinesArgs](
	"read_go_file_lines",
	"按指定行号范围读取工作区内的 Go 代码文件，输出带行号，便于对照编译错误信息。参数：directory（可选）、file_name（必填）、line_ranges（可选，如 \"10-12,20-30\" 表示第 10-12 行和第 20-30 行；不传则返回整个文件并带行号）。",
)

func (t *ReadGoFileLinesTool) Definition() dto.ToolDef {
	return readGoFileLinesToolDef
}

func (t *ReadGoFileLinesTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readGoFileLinesArgs](call.Args)
	if err != nil {
		return toolResult("read_go_file_lines 参数解析失败: "+err.Error(), true)
	}
	content, isError := runReadGoFileLinesTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runReadGoFileLinesTool 按行号范围读取工作区代码文件，输出带行号（便于对照编译错误）
func runReadGoFileLinesTool(ctx context.Context, args readGoFileLinesArgs, currentFullCodePath string) (string, bool) {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	fileName := strings.TrimSpace(args.FileName)
	lineRangesStr := strings.TrimSpace(args.LineRanges)

	if fileName == "" {
		return "read_go_file_lines 需传 file_name。", true
	}

	if prompt.IsPromptDocPath(targetPath) {
		return "read_go_file_lines 仅用于工作区代码文件，不能读文档路径；请用 read_doc 读取文档。", true
	}

	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}
	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	var matched *dto.WorkspaceContextFile
	for i := range workspaceCtx.Files {
		f := &workspaceCtx.Files[i]
		if f.FileName == fileName || f.FileName+"."+f.FileType == fileName || f.RelativePath == fileName {
			matched = f
			break
		}
	}
	if matched == nil {
		return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
	}

	lines := strings.Split(matched.Content, "\n")
	totalLines := len(lines)
	if totalLines > 0 && lines[totalLines-1] == "" {
		totalLines--
	}

	ranges := parseLineRanges(lineRangesStr, totalLines)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("文件 %s（目录：%s）共 %d 行\n\n", matched.RelativePath, targetPath, totalLines))

	width := 1
	for n := totalLines; n >= 10; n /= 10 {
		width++
	}

	for idx, rng := range ranges {
		for i := rng.Start; i <= rng.End && i <= totalLines; i++ {
			lineContent := ""
			if i-1 < len(lines) {
				lineContent = lines[i-1]
			}
			sb.WriteString(fmt.Sprintf("%*d | %s\n", width, i, lineContent))
		}
		if len(ranges) > 1 && idx < len(ranges)-1 {
			sb.WriteString("...\n")
		}
	}

	return sb.String(), false
}

// lineRange 行号范围，1-based 包含两端
type lineRange struct{ Start, End int }

// parseLineRanges 解析 "10-12,20-30" 或 "10,20-22"；空字符串表示全文，返回 []{1, totalLines}
func parseLineRanges(s string, totalLines int) []lineRange {
	s = strings.TrimSpace(s)
	if totalLines <= 0 {
		totalLines = 1
	}
	if s == "" {
		return []lineRange{{1, totalLines}}
	}
	var out []lineRange
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "-")
		if idx < 0 {
			var n int
			if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 {
				if n > totalLines {
					n = totalLines
				}
				out = append(out, lineRange{n, n})
			}
			continue
		}
		var start, end int
		if _, err := fmt.Sscanf(part, "%d-%d", &start, &end); err == nil && start >= 1 {
			if start > totalLines {
				start = totalLines
			}
			if end < start {
				end = start
			}
			if end > totalLines {
				end = totalLines
			}
			out = append(out, lineRange{start, end})
		}
	}
	if len(out) == 0 {
		return []lineRange{{1, totalLines}}
	}
	return out
}
