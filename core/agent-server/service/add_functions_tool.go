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
	return formatWriteGoFileResult(fileName, buildWorkspace, resp), false
}

// formatWriteGoFileResult 根据是否编译、以及编译/变更信息，返回对用户友好的提示
func formatWriteGoFileResult(fileName string, buildWorkspace bool, resp *dto.AddFunctionsResp) string {
	if !buildWorkspace {
		return fmt.Sprintf("已落盘: %s。当前未编译工作空间，仅修改了代码。改完后请调用 build_workspace 更新工作空间。", fileName)
	}
	// 已编译
	msg := fmt.Sprintf("已落盘并已编译部署: %s", fileName)
	if resp.BuildNewVersion != "" {
		if resp.BuildOldVersion != "" {
			msg += fmt.Sprintf("；版本 %s → %s", resp.BuildOldVersion, resp.BuildNewVersion)
		} else {
			msg += fmt.Sprintf("；当前版本 %s", resp.BuildNewVersion)
		}
	}
	if len(resp.BuildDiffAdd) > 0 {
		msg += "；新增接口: " + strings.Join(resp.BuildDiffAdd, ", ")
	}
	if len(resp.BuildDiffUpdate) > 0 {
		msg += "；变更接口: " + strings.Join(resp.BuildDiffUpdate, ", ")
	}
	if len(resp.BuildDiffDelete) > 0 {
		msg += "；删除接口: " + strings.Join(resp.BuildDiffDelete, ", ")
	}
	return msg + "。"
}
