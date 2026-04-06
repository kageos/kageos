package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type BuildWorkspaceTool struct{}

type buildWorkspaceArgs struct{}

var buildWorkspaceToolDef = toolDefinition[buildWorkspaceArgs](
	"build_workspace",
	"编译当前工作空间（Go 应用）。不写文件，仅基于当前已落盘的代码触发一次编译并部署。无需传参。连续写多个文件后可调用一次 build_workspace 再编译。",
)

func (t *BuildWorkspaceTool) Definition() dto.ToolDef {
	return buildWorkspaceToolDef
}

func (t *BuildWorkspaceTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	content, isError := runBuildWorkspaceTool(ctx, call.FullCodePath)
	return toolResult(content, isError)
}

// runBuildWorkspaceTool 编译当前工作空间（不写文件，仅触发编译并部署）；从当前工作目录解析 user/app，无需参数
func runBuildWorkspaceTool(ctx context.Context, currentFullCodePath string) (string, bool) {
	dir := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if dir == "" {
		return "build_workspace 无法获取当前工作目录，请确保在有效的工作台会话中操作", true
	}
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		return "build_workspace 当前目录格式应为 /user/app 或更长路径（如 /luobei/demo）", true
	}
	user, app := parts[0], parts[1]
	resp, err := apicall.UpdateAppBuild(ctx, user, app)
	if err != nil {
		logger.Errorf(ctx, "[WorkspaceBuild] UpdateAppBuild 失败: %v", err)
		return "build_workspace 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("工作空间已编译并部署: app=%s, 旧版本=%s, 新版本=%s", resp.App, resp.OldVersion, resp.NewVersion), false
}
