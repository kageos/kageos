package service

import "github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"

func workspaceToolNamesForMode(modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string) []string {
	var toolNames []string
	if modeProvider != nil {
		toolNames = modeProvider.ToolNames()
	} else {
		toolNames = fallbackToolNames
	}
	return append([]string(nil), toolNames...)
}
