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

func workspaceModeToolGateResult(_ string, _ []string) (ToolResult, bool) {
	// Mode config controls which tools are exposed to the model. Once a tool
	// call reaches the executor, the mode layer does not add another hard
	// runtime block.
	return ToolResult{}, false
}
