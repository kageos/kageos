package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// RunAddFunctionsTool 内部工具：将源码落盘到 directory（full_code_path）对应目录
// args: file_name, source_code（必填）；build_workspace 可选，默认 true；User 从 ctx 取
// buildWorkspace=false 时仅写文件不编译（对应 SkipBuild=true）
func RunAddFunctionsTool(ctx context.Context, args map[string]interface{}, fullCodePath string, skipMetadataParse bool, buildWorkspace bool) (content string, isError bool) {
	fileName := GetStringArg(args, "file_name")
	sourceCode := GetStringArg(args, "source_code")
	if fileName == "" || sourceCode == "" {
		return "write_go_file 缺少 file_name 或 content/source_code", true
	}
	if strings.TrimSpace(fullCodePath) == "" {
		return "write_go_file 需要 directory（当前目录）", true
	}

	user := contextx.GetRequestUser(ctx)
	if user == "" {
		logger.Warnf(ctx, "[AddFunctionsTool] RequestUser 为空")
		user = "system"
	}

	req := &dto.AddFunctionsReq{
		FullCodePath: strings.TrimSpace(fullCodePath),
		User:         user,
		FileName:     fileName,
		SourceCode:   sourceCode,
		SkipBuild:    !buildWorkspace, // build_workspace=false => 只写不编译
	}

	resp, err := apicall.ServiceTreeAddFunctions(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[AddFunctionsTool] ServiceTreeAddFunctions 失败: %v", err)
		return "write_go_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "write_go_file 失败: " + resp.Error, true
	}
	return fmt.Sprintf("已落盘: app_id=%d, app_code=%s, file_name=%s", resp.AppID, resp.AppCode, fileName), false
}
