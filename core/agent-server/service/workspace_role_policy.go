package service

import (
	"fmt"
	"strings"
)

func workspaceRoleToolGateResult(roleID string, toolName string) (ToolResult, bool) {
	toolName = normalizeWorkspaceToolName(toolName)
	if toolName == "" {
		return toolResult("工具名为空，无法执行。", true), true
	}
	if isWorkspaceReadOnlyTool(toolName) {
		return ToolResult{}, false
	}
	roleID = normalizeWorkspaceRole(roleID)
	if roleID == "" {
		if containsWorkspaceRoleString([]string{"change_role", "read_doc", "read_dir", "summarize_task_state"}, toolName) {
			return ToolResult{}, false
		}
		return toolResult(fmt.Sprintf(
			"当前还没有明确工作台角色，不能调用 %s。请先调用 change_role 选择产品经理、应用开发工程师、应用操作员、测试工程师等角色。",
			toolName,
		), true), true
	}
	spec, ok := workspaceRoleSpecFor(roleID)
	if !ok {
		return toolResult(fmt.Sprintf(
			"当前工作台角色 %q 不存在，不能调用 %s。请先调用 change_role 重新选择有效角色。",
			roleID, toolName,
		), true), true
	}
	if containsWorkspaceRoleString(spec.ForbiddenTools, toolName) {
		return toolResult(fmt.Sprintf(
			"当前角色「%s」不能调用 %s。%s",
			spec.DisplayName, toolName, workspaceRoleGateSuggestion(roleID, toolName),
		), true), true
	}
	if containsWorkspaceRoleString(spec.AllowedTools, toolName) {
		return ToolResult{}, false
	}
	return toolResult(fmt.Sprintf(
		"当前角色「%s」未授权调用 %s。%s",
		spec.DisplayName, toolName, workspaceRoleGateSuggestion(roleID, toolName),
	), true), true
}

func normalizeWorkspaceToolName(toolName string) string {
	return strings.TrimSpace(toolName)
}

func isWorkspaceReadOnlyTool(toolName string) bool {
	return containsWorkspaceRoleString(workspaceRoleBaseReadOnlyTools(), toolName)
}

func workspaceRoleBaseReadOnlyTools() []string {
	return []string{
		"read_doc",
		"read_dir",
		"read_go_file",
		"read_go_file_lines",
		"read_app_log",
		"search_tools",
		"search_resources",
		"summarize_task_state",
	}
}

func workspaceRoleGateSuggestion(roleID string, toolName string) string {
	switch toolName {
	case "write_prd":
		return "如需重新设计 PRD，请交接给「产品经理」。"
	case "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace":
		return "如需创建或修改代码，请交接给「应用开发工程师」「应用维护工程师」或「构建修复工程师」。"
	case "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query", "run_on_select_fuzzy":
		return "如需执行业务操作，请交接给「应用操作员」；如需验证功能，请交接给「测试工程师」。"
	default:
		if roleID == WorkspaceRoleProductManager {
			return "产品经理只负责需求分析、PRD 和确认。"
		}
		return "请先切换到有该工具权限的角色。"
	}
}

func workspaceRoleFromToolResult(result ToolResult) string {
	if result.IsError || result.Data == nil {
		return ""
	}
	switch data := result.Data.(type) {
	case changeRoleData:
		if data.RoleID != "" {
			return normalizeWorkspaceRole(data.RoleID)
		}
		return normalizeWorkspaceRole(data.CurrentRole)
	case *changeRoleData:
		if data == nil {
			return ""
		}
		if data.RoleID != "" {
			return normalizeWorkspaceRole(data.RoleID)
		}
		return normalizeWorkspaceRole(data.CurrentRole)
	case map[string]interface{}:
		if role, _ := data["role_id"].(string); role != "" {
			return normalizeWorkspaceRole(role)
		}
		if role, _ := data["current_role"].(string); role != "" {
			return normalizeWorkspaceRole(role)
		}
	}
	return ""
}
