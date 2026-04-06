package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type WriteGoFileTool struct{}

type writeGoFileArgs struct {
	Directory    string `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"Go 文件名" schema_required:"true"`
	Content      string `json:"content" schema_desc:"Go 源码全文" schema_required:"true"`
	SourceCode   string `json:"source_code" schema_ignore:"true"`
}

var writeGoFileToolDef = toolDefinition[writeGoFileArgs](
	"write_go_file",
	"在当前工作目录或指定 directory 下写入一个 .go 代码文件。必填：file_name（如 attendance.go）、content（Go 源码）。可选：directory（目标目录）。注意：write_go_file 只落盘、不编译；可连续多次写入多个文件，全部写完后统一调用一次 build_workspace 完成编译与部署，无需每写一次就编译。",
)

func (t *WriteGoFileTool) Definition() dto.ToolDef {
	return writeGoFileToolDef
}

func (t *WriteGoFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[writeGoFileArgs](call.Args)
	if err != nil {
		return toolResult("write_go_file 参数解析失败: "+err.Error(), true)
	}
	content, isError := runWriteGoFileTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runWriteGoFileTool 写 Go 代码文件；固定仅落盘，不触发编译
func runWriteGoFileTool(ctx context.Context, args writeGoFileArgs, currentFullCodePath string) (string, bool) {
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return "write_go_file 缺少参数 file_name。", true
	}
	content := args.Content
	if content == "" {
		content = args.SourceCode
	}
	if content == "" {
		return "write_go_file 缺少参数 content。", true
	}
	if !strings.HasSuffix(fileName, ".go") {
		fileName += ".go"
	}
	if strings.TrimSuffix(fileName, ".go") == "init_" {
		return "不允许创建该文件，由脚手架自动生成。", true
	}

	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	return runAddFunctionsCommand(ctx, addFunctionsCommand{
		FileName:   fileName,
		SourceCode: content,
	}, targetPath, false)
}
