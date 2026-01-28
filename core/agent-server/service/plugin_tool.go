package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// RunPluginTool 插件工具：CallFormAPI(FormPath, {Content, InputFiles})
// args: content（可选）；files 由 WorkspaceChat 从 message.files 注入，可为 nil
func RunPluginTool(ctx context.Context, p *model.Plugin, args map[string]interface{}, files *types.Files) (content string, isError bool) {
	formPath := strings.TrimSpace(p.FormPath)
	if formPath == "" {
		return "插件 FormPath 为空: " + p.Code, true
	}

	req := &dto.AgentPluginFormReq{
		Content:    GetStringArg(args, "content"),
		InputFiles: files,
	}

	resp, err := apicall.CallFormAPI[dto.AgentPluginFormReq, *dto.AgentPluginFormResp](ctx, formPath, *req)
	if err != nil {
		logger.Errorf(ctx, "[PluginTool] CallFormAPI 失败: FormPath=%s, plugin=%s, err=%v", formPath, p.Code, err)
		return "插件调用失败: " + err.Error(), true
	}
	if resp == nil {
		return "插件返回为空", true
	}
	return resp.Result, false
}
