package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type addFunctionsCommand struct {
	FileName   string
	SourceCode string
}

// runAddFunctionsCommand 将源码落盘到 full_code_path 对应目录。
// 租户由 app-server 从 full_code_path 解析，不传 User；buildWorkspace=false 时仅写文件不编译（对应 SkipBuild=true）。
func runAddFunctionsCommand(ctx context.Context, cmd addFunctionsCommand, fullCodePath string, buildWorkspace bool) (content string, isError bool) {
	fileName := strings.TrimSpace(cmd.FileName)
	sourceCode := cmd.SourceCode
	if fileName == "" || sourceCode == "" {
		return "write_go_file 缺少 file_name 或 content/source_code", true
	}
	if strings.TrimSpace(fullCodePath) == "" {
		return "write_go_file 需要 directory（当前目录）", true
	}

	// 租户由 app-server 从 full_code_path 解析出的目录所属应用确定，不传 User
	req := &dto.AddFunctionsReq{
		FullCodePath: strings.TrimSpace(fullCodePath),
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
