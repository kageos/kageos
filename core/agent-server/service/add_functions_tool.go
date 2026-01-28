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

// RunAddFunctionsTool 内部工具：将源码落盘到 full_code_path 对应目录
// args: file_name, source_code（必填）；User 从 ctx 取
// 注意：此函数供 generate_code 工具和内部 write_file 工具调用，不直接暴露给大模型
func RunAddFunctionsTool(ctx context.Context, args map[string]interface{}, fullCodePath string) (content string, isError bool) {
	fileName := GetStringArg(args, "file_name")
	sourceCode := GetStringArg(args, "source_code")
	if fileName == "" || sourceCode == "" {
		return "write_file 缺少 file_name 或 content/source_code", true
	}
	if strings.TrimSpace(fullCodePath) == "" {
		return "write_file 需要 full_code_path（当前目录）", true
	}

	user := contextx.GetRequestUser(ctx)
	if user == "" {
		logger.Warnf(ctx, "[AddFunctionsTool] RequestUser 为空")
		user = "system"
	}

	// 工作台场景：只传必要参数，其他字段使用默认值（0 或 false）
	req := &dto.AddFunctionsReq{
		FullCodePath: strings.TrimSpace(fullCodePath),
		// RecordID, MessageID, AgentID 在工作台场景下不需要，使用默认值 0
		// Async 在工作台场景下不需要，使用默认值 false
		// User 从 ctx 获取，但需要传入以便 app-server 使用
		User:       user,
		FileName:   fileName,
		SourceCode: sourceCode,
	}

	resp, err := apicall.ServiceTreeAddFunctions(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[AddFunctionsTool] ServiceTreeAddFunctions 失败: %v", err)
		return "write_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "write_file 失败: " + resp.Error, true
	}
	return fmt.Sprintf("已落盘: app_id=%d, app_code=%s, file_name=%s", resp.AppID, resp.AppCode, fileName), false
}
