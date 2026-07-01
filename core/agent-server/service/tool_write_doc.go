package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/dto"
)

type WriteDocTool struct{}

type writeDocArgs struct {
	Directory    string `json:"directory" schema_desc:"父目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	Name         string `json:"name" schema_desc:"文档显示名称" schema_required:"true"`
	Code         string `json:"code" schema_desc:"文档英文标识" schema_required:"true"`
	Content      string `json:"content" schema_desc:"文档正文" schema_required:"true"`
	Format       string `json:"format" schema_desc:"文档格式，默认 markdown"`
}

var writeDocToolDef = toolDefinition[writeDocArgs](
	"write_doc",
	"在指定目录下创建或更新一篇文档。必填：name（显示名称）、code（英文标识）、content（正文）。可选：directory（父目录，不传则当前工作目录）、format（默认 markdown）。写当前目录运行手册时使用 code=runbook、name=运行手册，生成 <当前目录>/runbook.docs；该文档会作为当前目录业务背景/SOP 自动注入工作台上下文。",
)

func (t *WriteDocTool) Definition() dto.ToolDef {
	return writeDocToolDef
}

func (t *WriteDocTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[writeDocArgs](call.Args)
	if err != nil {
		return toolResult("write_doc 参数解析失败: "+err.Error(), true)
	}
	content, isError := runWriteDocTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runWriteDocTool 写文档：目录 + name + code + content，内部转成共享 command 执行
func runWriteDocTool(ctx context.Context, args writeDocArgs, currentFullCodePath string) (string, bool) {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	return runWriteDocCommand(ctx, writeDocCommand{
		FullCodePath: targetPath,
		Name:         args.Name,
		Code:         args.Code,
		Content:      args.Content,
		Format:       args.Format,
	}, currentFullCodePath)
}
