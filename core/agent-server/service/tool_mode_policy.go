package service

import "github.com/kageos/kageos/core/agent-server/prompt"

func workspaceToolNamesForMode(modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string) []string {
	var toolNames []string
	if modeProvider != nil {
		toolNames = modeProvider.ToolNames()
	} else {
		toolNames = fallbackToolNames
	}
	return append([]string(nil), toolNames...)
}

func workspaceToolNamesForRole(toolNames []string, roleID string) []string {
	roleID = normalizeWorkspaceRole(roleID)
	if roleID == "" {
		roleID = WorkspaceRoleRouter
	}
	allowedTools := workspaceRoleAllowedTools(roleID)
	allowed := make(map[string]struct{}, len(allowedTools))
	for _, tool := range allowedTools {
		tool = normalizeWorkspaceToolName(tool)
		if tool != "" {
			allowed[tool] = struct{}{}
		}
	}
	out := make([]string, 0, len(toolNames))
	for _, tool := range toolNames {
		tool = normalizeWorkspaceToolName(tool)
		if tool == "" {
			continue
		}
		if _, ok := allowed[tool]; ok {
			out = append(out, tool)
		}
	}
	return out
}

func workspaceToolNamesForLLM(toolNames []string) []string {
	seen := make(map[string]struct{}, len(toolNames))
	out := make([]string, 0, len(toolNames))
	for _, tool := range toolNames {
		tool = normalizeWorkspaceToolName(tool)
		if tool == "" {
			continue
		}
		if _, ok := seen[tool]; ok {
			continue
		}
		seen[tool] = struct{}{}
		out = append(out, tool)
	}
	return out
}
