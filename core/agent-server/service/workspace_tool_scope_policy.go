package service

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

type workspaceToolScope struct {
	ExecuteDirectory   string
	TargetAppDirectory string
}

func workspaceToolScopeGateResult(roleID string, toolName string, args map[string]interface{}, executeDirectory string) (ToolResult, bool) {
	return workspaceToolScopeGateResultWithScope(roleID, toolName, args, workspaceToolScope{ExecuteDirectory: executeDirectory})
}

func workspaceToolScopeGateResultWithScope(roleID string, toolName string, args map[string]interface{}, toolScope workspaceToolScope) (ToolResult, bool) {
	toolName = normalizeWorkspaceToolName(toolName)
	if !workspaceToolUsesWorkbenchPath(toolName) {
		return ToolResult{}, false
	}
	scope := normalizeWorkspacePath(firstNonEmptyString(toolScope.ExecuteDirectory, toolScope.TargetAppDirectory))
	if scope == "" {
		return toolResult(fmt.Sprintf("%s 需要明确的 execute_directory；请先调用 change_role 固定执行目录。", toolName), true), true
	}
	if workspaceToolRequiresScopedSearchDirectory(roleID, toolName, args) {
		return toolResult(fmt.Sprintf("%s 在当前角色下必须显式传 directory=change_role.execute_directory，或使用 scope=current_app；不要搜索整个工作空间。", toolName), true), true
	}
	targetScope := workspaceToolTargetAppScope(roleID, toolName, toolScope)
	if targetScope != "" {
		if res, blocked := workspaceTargetAppScopeGateResult(toolName, args, scope, targetScope); blocked {
			return res, true
		}
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

func workspaceToolTargetAppScope(roleID string, toolName string, toolScope workspaceToolScope) string {
	target := normalizeWorkspacePath(toolScope.TargetAppDirectory)
	execute := normalizeWorkspacePath(toolScope.ExecuteDirectory)
	if target == "" || target == execute {
		return ""
	}
	if !workspaceToolRequiresTargetAppDirectory(roleID, toolName) && normalizeWorkspaceToolName(toolName) != "create_directory" {
		return ""
	}
	return target
}

func workspaceTargetAppScopeGateResult(toolName string, args map[string]interface{}, executeDirectory string, targetAppDirectory string) (ToolResult, bool) {
	toolName = normalizeWorkspaceToolName(toolName)
	targetAppDirectory = normalizeWorkspacePath(targetAppDirectory)
	if targetAppDirectory == "" {
		return ToolResult{}, false
	}
	if toolName == "create_directory" {
		createdPath := workspaceCreateDirectoryTargetPath(args, executeDirectory)
		if createdPath == "" || (createdPath != targetAppDirectory && !workspacePathHasPrefix(createdPath, targetAppDirectory)) {
			return toolResult(fmt.Sprintf(
				"create_directory 被目录落点门禁阻断：本阶段目标应用目录是 %s。请在父目录 %s 下创建 code=%q；不要在当前目录随意创建其他目录。",
				targetAppDirectory,
				executeDirectory,
				workspaceTargetDirectoryCode(targetAppDirectory),
			), true), true
		}
		return ToolResult{}, false
	}
	if toolName == "build_workspace" && normalizeWorkspacePath(executeDirectory) != targetAppDirectory {
		return toolResult(fmt.Sprintf(
			"build_workspace 被目录落点门禁阻断：本阶段目标应用目录是 %s，当前 execute_directory 是 %s。请先把执行目录切到目标应用目录，确认代码都写在目标目录后再构建。",
			targetAppDirectory,
			executeDirectory,
		), true), true
	}
	candidates := workspaceToolScopeCandidatePaths(toolName, args)
	if len(candidates) == 0 && workspaceToolDefaultsToExecuteDirectory(toolName) {
		candidates = []string{executeDirectory}
	}
	for _, candidate := range candidates {
		path := normalizeWorkspacePath(candidate)
		if path == "" || workspaceToolPathBypassesScope(toolName, path) {
			continue
		}
		if !workspacePathHasPrefix(path, targetAppDirectory) {
			return toolResult(fmt.Sprintf(
				"%s 被目录落点门禁阻断：本阶段目标应用目录是 %s，请把 directory/full_code_path 限定到该目录内；不要写入或操作父目录 %s。",
				toolName,
				targetAppDirectory,
				executeDirectory,
			), true), true
		}
	}
	return ToolResult{}, false
}

func workspaceToolRequiresTargetAppDirectory(roleID string, toolName string) bool {
	switch normalizeWorkspaceToolName(toolName) {
	case "write_go_file",
		"search_replace_file",
		"delete_file",
		"build_workspace",
		"check_workspace_code",
		"run_table_search",
		"run_table_create",
		"run_table_update",
		"run_table_delete",
		"run_form_submit",
		"run_chart_query",
		"run_on_select_fuzzy":
		return true
	default:
		return false
	}
}

func workspaceToolDefaultsToExecuteDirectory(toolName string) bool {
	switch normalizeWorkspaceToolName(toolName) {
	case "write_go_file", "search_replace_file", "delete_file", "check_workspace_code":
		return true
	default:
		return false
	}
}

func workspaceCreateDirectoryTargetPath(args map[string]interface{}, executeDirectory string) string {
	if args == nil {
		return ""
	}
	parent := firstWorkspaceToolStringArg(args, "directory", "full_code_path")
	parent = normalizeWorkspacePath(firstNonEmptyString(parent, executeDirectory))
	code := strings.Trim(firstWorkspaceToolStringArg(args, "code"), "/")
	if parent == "" || code == "" {
		return ""
	}
	return normalizeWorkspacePath(parent + "/" + code)
}

func firstWorkspaceToolStringArg(args map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := args[key].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func workspaceTargetDirectoryCode(targetAppDirectory string) string {
	targetAppDirectory = normalizeWorkspacePath(targetAppDirectory)
	if targetAppDirectory == "" {
		return ""
	}
	parts := workspacePathParts(targetAppDirectory)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
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
		"build_workspace",
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
