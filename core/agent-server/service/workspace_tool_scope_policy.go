package service

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

func workspaceToolScopeGateResult(roleID string, toolName string, args map[string]interface{}, executeDirectory string) (ToolResult, bool) {
	toolName = normalizeWorkspaceToolName(toolName)
	if !workspaceToolUsesWorkbenchPath(toolName) {
		return ToolResult{}, false
	}
	scope := normalizeWorkspacePath(executeDirectory)
	if scope == "" {
		return toolResult(fmt.Sprintf("%s 需要明确的 execute_directory；请先调用 change_role 固定执行目录。", toolName), true), true
	}
	if workspaceToolRequiresScopedSearchDirectory(roleID, toolName, args) {
		return toolResult(fmt.Sprintf("%s 在当前角色下必须显式传 directory=change_role.execute_directory，或使用 scope=current_app；不要搜索整个工作空间。", toolName), true), true
	}
	for _, candidate := range workspaceToolScopeCandidatePaths(toolName, args) {
		path := normalizeWorkspacePath(candidate)
		if path == "" || workspaceToolPathBypassesScope(toolName, path) {
			continue
		}
		if !workspacePathHasPrefix(path, scope) {
			return toolResult(fmt.Sprintf(
				"%s 被目录作用域门禁阻断：请求路径 %s 不在当前 execute_directory %s 内。请先 change_role 到正确目录，或把工具参数限定到当前执行目录。",
				toolName,
				path,
				scope,
			), true), true
		}
	}
	return ToolResult{}, false
}

func workspaceToolUsesWorkbenchPath(toolName string) bool {
	switch normalizeWorkspaceToolName(toolName) {
	case "read_doc",
		"read_dir",
		"read_go_file",
		"read_go_file_lines",
		"read_app_log",
		"search_tools",
		"search_resources",
		"create_directory",
		"write_go_file",
		"search_replace_file",
		"delete_file",
		"run_table_search",
		"run_table_create",
		"run_table_update",
		"run_table_delete",
		"run_form_submit",
		"run_chart_query",
		"run_on_select_fuzzy",
		"check_workspace_code":
		return true
	default:
		return false
	}
}

func workspaceToolRequiresScopedSearchDirectory(roleID string, toolName string, args map[string]interface{}) bool {
	switch normalizeWorkspaceToolName(toolName) {
	case "search_tools", "search_resources":
	default:
		return false
	}
	switch normalizeWorkspaceRole(roleID) {
	case WorkspaceRoleAppOperator, WorkspaceRoleQAEngineer:
	default:
		return false
	}
	if s, _ := args["directory"].(string); strings.TrimSpace(s) != "" {
		return false
	}
	if s, _ := args["scope"].(string); strings.TrimSpace(s) == "current_app" {
		return false
	}
	return true
}

func workspaceToolScopeCandidatePaths(toolName string, args map[string]interface{}) []string {
	keys := []string{"directory", "full_code_path"}
	if normalizeWorkspaceToolName(toolName) == "read_doc" {
		keys = []string{"directory"}
	}
	out := []string{}
	for _, key := range keys {
		out = append(out, workspaceToolScopePathsFromArg(args[key])...)
	}
	return out
}

func workspaceToolScopePathsFromArg(raw interface{}) []string {
	value, ok := raw.(string)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func workspaceToolPathBypassesScope(toolName string, path string) bool {
	path = normalizeWorkspacePath(path)
	if path == "" {
		return true
	}
	if prompt.IsPromptDocPath(path) {
		return true
	}
	return normalizeWorkspaceToolName(toolName) == "read_doc" && strings.HasPrefix(path, "/system/prompt/")
}
