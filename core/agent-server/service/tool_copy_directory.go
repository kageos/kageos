package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type CopyDirectoryTool struct{}

type copyDirectoryArgs struct {
	SourceDirectory string `json:"source_directory" schema_desc:"源目录完整路径" schema_required:"true"`
	TargetDirectory string `json:"target_directory" schema_desc:"目标父目录" schema_required:"true"`
}

var copyDirectoryToolDef = toolDefinition[copyDirectoryArgs](
	"copy_directory",
	"将本地目录复制到工作区。source_directory 填本地完整路径（如 /user/app/plugins/xxx）。target_directory 填目标父目录即当前工作区路径（如 /luobei/myapp/server），不要填「父目录+子目录名」；系统会在该父目录下自动创建与源同名的目录（如源为 .../video_tools 则得到 .../server/video_tools）。复制成功后会自动编译，无需再调用 build_workspace；返回目录数、文件数。",
)

func (t *CopyDirectoryTool) Definition() dto.ToolDef {
	return copyDirectoryToolDef
}

func (t *CopyDirectoryTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[copyDirectoryArgs](call.Args)
	if err != nil {
		return toolResult("copy_directory 参数解析失败: "+err.Error(), true)
	}
	content, isError := runCopyDirectoryTool(ctx, args)
	return toolResult(content, isError)
}

func runCopyDirectoryTool(ctx context.Context, args copyDirectoryArgs) (string, bool) {
	sourcePath := strings.TrimSpace(args.SourceDirectory)
	if sourcePath == "" {
		return "copy_directory 必填 source_directory（本地完整路径如 /user/app/plugins/xxx）。", true
	}
	if strings.HasPrefix(sourcePath, "hub://") {
		return "copy_directory 不再支持 hub:// 链接；请使用本地目录路径或能力包导入。", true
	}
	if !strings.HasPrefix(sourcePath, "/") {
		sourcePath = "/" + sourcePath
	}
	targetPath := strings.TrimSpace(args.TargetDirectory)
	if targetPath == "" {
		return "copy_directory 必填 target_directory（目标父目录，即当前工作区路径，如 /user/app/server；会在其下创建与源同名的子目录，不要填 .../子目录名）。", true
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	pathForDetail := targetPath
	for {
		detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, pathForDetail)
		if err == nil && detail != nil && detail.AppID > 0 {
			req := &dto.CopyDirectoryReq{
				SourceDirectoryPath: sourcePath,
				TargetDirectoryPath: targetPath,
				TargetAppID:         detail.AppID,
			}
			resp, err := apicall.CopyDirectoryViaWorkspace(ctx, req)
			if err != nil {
				logger.Errorf(ctx, "[CopyDirectory] CopyDirectory 失败: %v", err)
				return "copy_directory 复制失败: " + err.Error(), true
			}
			return fmt.Sprintf("复制成功。%s（目录数: %d，文件数: %d）", resp.Message, resp.DirectoryCount, resp.FileCount), false
		}
		idx := strings.LastIndex(strings.Trim(pathForDetail, "/"), "/")
		if idx <= 0 {
			break
		}
		pathForDetail = "/" + strings.Trim(pathForDetail, "/")[:idx]
		if pathForDetail == "" || pathForDetail == "/" {
			break
		}
	}
	return "copy_directory: 无法解析目标应用（target_directory 为目标父目录，须为工作区已存在路径，如 /user/app/server；不要填 .../子目录名）。", true
}
