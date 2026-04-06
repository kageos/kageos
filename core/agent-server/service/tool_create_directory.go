package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type CreateDirectoryTool struct{}

type createDirectoryArgs struct {
	Directory    string `json:"directory" schema_desc:"父目录，不传则使用当前目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	Name         string `json:"name" schema_desc:"目录显示名称" schema_required:"true"`
	Code         string `json:"code" schema_desc:"目录代码标识" schema_required:"true"`
	Description  string `json:"description" schema_desc:"目录描述"`
	Tags         string `json:"tags" schema_desc:"标签，逗号分隔"`
	Admins       string `json:"admins" schema_desc:"管理员列表，逗号分隔"`
}

var createDirectoryToolDef = toolDefinition[createDirectoryArgs](
	"create_directory",
	"在当前目录或指定 directory（父目录）下创建一个子目录（package 类型）。必填：name（显示名称）、code（代码标识）。可选：directory（父目录）、description、tags、admins。",
)

func (t *CreateDirectoryTool) Definition() dto.ToolDef {
	return createDirectoryToolDef
}

func (t *CreateDirectoryTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[createDirectoryArgs](call.Args)
	if err != nil {
		return toolResult("create_directory 参数解析失败: "+err.Error(), true)
	}
	return toolResult(runCreateDirectoryCommand(ctx, createDirectoryCommand{
		Directory:    args.Directory,
		FullCodePath: args.FullCodePath,
		Name:         args.Name,
		Code:         args.Code,
		Description:  args.Description,
		Tags:         args.Tags,
		Admins:       args.Admins,
	}, call.FullCodePath))
}
