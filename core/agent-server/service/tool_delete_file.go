package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type DeleteFileTool struct{}

type deleteFileArgs struct {
	Directory    string `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"要删除的文件名" schema_required:"true"`
}

var deleteFileToolDef = toolDefinition[deleteFileArgs](
	"delete_file",
	"删除指定目录下的一个 .go 代码文件。必填：directory（或当前目录）、file_name。会同时删除磁盘文件和 DB 节点。不能删除 init_.go。",
)

func (t *DeleteFileTool) Definition() dto.ToolDef {
	return deleteFileToolDef
}

func (t *DeleteFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[deleteFileArgs](call.Args)
	if err != nil {
		return toolResult("delete_file 参数解析失败: "+err.Error(), true)
	}
	content, isError := runDeleteFileTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runDeleteFileTool 删除目录下指定 .go 文件（删磁盘+删节点）
func runDeleteFileTool(ctx context.Context, args deleteFileArgs, currentFullCodePath string) (string, bool) {
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
		return "delete_file 缺少参数 file_name。", true
	}
	req := &dto.DeleteFileReq{
		FullCodePath: targetPath,
		FileName:     fileName,
	}
	resp, err := apicall.DeleteFile(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[DeleteFile] DeleteFile 失败: %v", err)
		return "delete_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "delete_file: " + resp.Message, true
	}
	return fmt.Sprintf("已删除: 目录=%s, 文件=%s", targetPath, fileName), false
}
