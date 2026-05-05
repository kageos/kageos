package service

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
)

func workspaceToolNamesForMode(modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string) []string {
	var toolNames []string
	if modeProvider != nil {
		toolNames = modeProvider.ToolNames()
	} else {
		toolNames = fallbackToolNames
	}
	return appendWorkspaceSkillToolNames(toolNames)
}

func workspaceModeToolGateResult(toolName string, allowedToolNames []string) (ToolResult, bool) {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" || len(allowedToolNames) == 0 {
		return ToolResult{}, false
	}
	for _, allowed := range allowedToolNames {
		if toolName == strings.TrimSpace(allowed) {
			return ToolResult{}, false
		}
	}
	return toolResult(fmt.Sprintf("当前工作台模式不允许调用工具 `%s`。请切换到包含该工具的模式，或改用当前模式允许的只读/执行工具。", toolName), true), true
}
