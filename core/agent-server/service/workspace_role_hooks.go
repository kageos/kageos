package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/functionschema"
)

const (
	workspaceRoleHookStageBeforeHandoff        = "before_handoff"
	workspaceRoleHookStageBeforeEnter          = "before_enter"
	workspaceRoleHookProductManagerToDeveloper = "product_manager.to_app_developer"
	workspaceRoleHookAppOperatorCapabilities   = "app_operator.before_enter_capabilities"
	workspaceRoleHookBuildEngineerDiagnostics  = "build_engineer.before_enter_diagnostics"
)

var workspaceRoleHookSearchFunctions = apicall.SearchFunctions

type workspaceRoleHookInput struct {
	Stage              string
	SourceRole         string
	TargetRole         string
	ArtifactKind       string
	Artifact           map[string]interface{}
	FullCodePath       string
	WorkspaceDirectory string
	TargetAppDirectory string
	ExecuteDirectory   string
	Handoff            roleHandoffData
	Messages           []*model.AgentChatMessage
}

type workspaceRoleHookOutput struct {
	PRDExecutionMarkdown  string
	AppCapabilities       *workspaceAppCapabilitySnapshot
	BuildDiagnostics      *workspaceBuildDiagnostics
	ExecutedHooks         []workspaceExecutedRoleHook
	HandoffKeyInformation []string
}

type workspaceExecutedRoleHook struct {
	ID         string   `json:"id"`
	Stage      string   `json:"stage"`
	SourceRole string   `json:"source_role,omitempty"`
	TargetRole string   `json:"target_role,omitempty"`
	Status     string   `json:"status"`
	Produced   []string `json:"produced,omitempty"`
	Note       string   `json:"note,omitempty"`
}

type workspaceAppCapabilitySnapshot struct {
	Status             string                           `json:"status"`
	ExecuteDirectory   string                           `json:"execute_directory,omitempty"`
	Scope              string                           `json:"scope,omitempty"`
	User               string                           `json:"user,omitempty"`
	App                string                           `json:"app,omitempty"`
	Keyword            string                           `json:"keyword,omitempty"`
	TotalFunctions     int                              `json:"total_functions"`
	DisplayedFunctions int                              `json:"displayed_functions"`
	Counts             workspaceAppCapabilityCounts     `json:"counts"`
	Functions          []workspaceAppFunctionCapability `json:"functions,omitempty"`
	Guidance           []string                         `json:"guidance,omitempty"`
	Error              string                           `json:"error,omitempty"`
}

type workspaceAppCapabilityCounts struct {
	Tables int `json:"tables"`
	Forms  int `json:"forms"`
	Charts int `json:"charts"`
}

type workspaceAppFunctionCapability struct {
	Name          string   `json:"name,omitempty"`
	Code          string   `json:"code,omitempty"`
	FullCodePath  string   `json:"full_code_path,omitempty"`
	Type          string   `json:"type,omitempty"`
	Capabilities  string   `json:"capabilities,omitempty"`
	RunTools      []string `json:"run_tools,omitempty"`
	Description   string   `json:"description,omitempty"`
	SchemaSummary []string `json:"schema_summary,omitempty"`
}

func runWorkspaceRoleHooks(input workspaceRoleHookInput) workspaceRoleHookOutput {
	out := workspaceRoleHookOutput{}
	if shouldRunWorkspacePRDToDeveloperHook(input) {
		sourceRole := normalizeWorkspaceRole(input.SourceRole)
		markdown := renderWorkspacePRDExecutionMarkdown(input.Artifact, input.ExecuteDirectory, input.TargetAppDirectory)
		if strings.TrimSpace(markdown) != "" {
			out.PRDExecutionMarkdown = markdown
			note := "已根据 agent_app_prd JSON 生成开发执行视图；目标模型不接收来源会话完整历史。"
			if sourceRole == "" {
				note += " 来源角色未记录，按目标角色和产物类型兼容触发。"
			}
			out.ExecutedHooks = append(out.ExecutedHooks, workspaceExecutedRoleHook{
				ID:         workspaceRoleHookProductManagerToDeveloper,
				Stage:      workspaceRoleHookStageBeforeHandoff,
				SourceRole: sourceRole,
				TargetRole: WorkspaceRoleAppDeveloper,
				Status:     "ok",
				Produced:   []string{"PRD_EXECUTION_MARKDOWN"},
				Note:       note,
			})
		}
	}
	return out
}

func runWorkspaceRoleBeforeEnterHooks(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	out := workspaceRoleHookOutput{}
	if shouldRunWorkspaceBuildEngineerDiagnosticsHook(input) {
		diagnostics := buildWorkspaceDiagnostics(workspaceBuildErrorTextFromHandoff(input.Handoff), input.ExecuteDirectory)
		out.BuildDiagnostics = diagnostics
		out.HandoffKeyInformation = append(out.HandoffKeyInformation, workspaceBuildDiagnosticsHandoffLines(diagnostics)...)
		out.ExecutedHooks = append(out.ExecutedHooks, workspaceExecutedRoleHook{
			ID:         workspaceRoleHookBuildEngineerDiagnostics,
			Stage:      workspaceRoleHookStageBeforeEnter,
			SourceRole: normalizeWorkspaceRole(input.SourceRole),
			TargetRole: WorkspaceRoleBuildEngineer,
			Status:     firstNonEmptyString(diagnostics.Status, "empty"),
			Produced:   []string{"build_diagnostics", "required_docs", "repair_policy", "executed_hooks"},
			Note:       workspaceBuildDiagnosticsHookNote(diagnostics),
		})
	}
	if shouldRunWorkspaceAppOperatorCapabilitiesHook(input) {
		snapshot := buildWorkspaceAppOperatorCapabilitySnapshot(ctx, input)
		out.AppCapabilities = snapshot
		out.HandoffKeyInformation = workspaceAppCapabilityHandoffLines(snapshot)
		out.ExecutedHooks = append(out.ExecutedHooks, workspaceExecutedRoleHook{
			ID:         workspaceRoleHookAppOperatorCapabilities,
			Stage:      workspaceRoleHookStageBeforeEnter,
			SourceRole: normalizeWorkspaceRole(input.SourceRole),
			TargetRole: WorkspaceRoleAppOperator,
			Status:     firstNonEmptyString(snapshot.Status, "skipped"),
			Produced:   []string{"available_capabilities", "operation_schema_summary"},
			Note:       workspaceAppCapabilityHookNote(snapshot),
		})
	}
	return out
}

func shouldRunWorkspacePRDToDeveloperHook(input workspaceRoleHookInput) bool {
	if strings.TrimSpace(input.Stage) != workspaceRoleHookStageBeforeHandoff {
		return false
	}
	if strings.TrimSpace(input.ArtifactKind) != "agent_app_prd" {
		return false
	}
	if normalizeWorkspaceRole(input.TargetRole) != WorkspaceRoleAppDeveloper {
		return false
	}
	if len(input.Artifact) == 0 {
		return false
	}
	sourceRole := normalizeWorkspaceRole(input.SourceRole)
	return sourceRole == "" || sourceRole == WorkspaceRoleProductManager
}

func shouldRunWorkspaceAppOperatorCapabilitiesHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleAppOperator
}

func shouldRunWorkspaceBuildEngineerDiagnosticsHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleBuildEngineer
}

func workspaceBuildErrorTextFromHandoff(handoff roleHandoffData) string {
	parts := make([]string, 0, len(handoff.TaskContext)+len(handoff.KeyInformation)+len(handoff.References)+1)
	candidates := make([]string, 0, len(handoff.TaskContext)+len(handoff.KeyInformation)+len(handoff.References))
	candidates = append(candidates, handoff.TaskContext...)
	candidates = append(candidates, handoff.KeyInformation...)
	candidates = append(candidates, handoff.References...)
	for _, item := range candidates {
		item = strings.TrimSpace(item)
		if item == "" || strings.HasPrefix(item, "下一步建议：") || strings.HasPrefix(item, "失败处理建议：") {
			continue
		}
		parts = append(parts, item)
	}
	text := strings.Join(parts, "\n")
	if !workspaceBuildDiagnosticsHasErrorSignal(text) {
		return ""
	}
	if handoff.ExecuteDirectory != "" {
		parts = append(parts, "execute_directory="+handoff.ExecuteDirectory)
	}
	return strings.Join(parts, "\n")
}

func workspaceBuildDiagnosticsHasErrorSignal(text string) bool {
	for _, keyword := range []string{
		"build_workspace 失败",
		"app startup failed",
		"SDK schema compile failed",
		"schema decode failed",
		"failed to validate",
		"audit field",
		"requires options or OnSelectFuzzyMap",
		"unsupported widget",
		"undefined:",
		"redeclared in this block",
		"报错",
		"错误",
		"failed",
		"error",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func workspaceBuildDiagnosticsHandoffLines(diagnostics *workspaceBuildDiagnostics) []string {
	if diagnostics == nil {
		return nil
	}
	lines := []string{}
	if diagnostics.Status == "empty" {
		lines = append(lines, "构建诊断：未拿到完整 build_workspace 错误；下一步先读取完整失败输出，再按 router/字段/文件归类。")
	} else {
		scope := firstNonEmptyString(diagnostics.WorkspacePath, "未指定")
		category := firstNonEmptyString(strings.Join(diagnostics.Categories, "、"), "build_failure")
		router := firstNonEmptyString(strings.Join(diagnostics.Routers, "、"), "未解析到 router")
		lines = append(lines, fmt.Sprintf("构建诊断：范围=%s；错误类型=%s；涉及 router=%s。", scope, category, router))
	}
	if len(diagnostics.FieldIssues) > 0 {
		fieldParts := []string{}
		for i, issue := range diagnostics.FieldIssues {
			if i >= 4 {
				break
			}
			fieldParts = append(fieldParts, fmt.Sprintf("%s(%s): %s", issue.Field, issue.JSONName, issue.Message))
		}
		lines = append(lines, "字段问题摘要："+compactText(strings.Join(fieldParts, "；"), 260))
	}
	if len(diagnostics.SDKSymbols) > 0 {
		lines = append(lines, "未确认 SDK/API 符号："+strings.Join(trimRoleHandoffStrings(diagnostics.SDKSymbols, 6), "、"))
	}
	if len(diagnostics.RequiredDocs) > 0 {
		lines = append(lines, "构建修复必读资料："+strings.Join(diagnostics.RequiredDocs, "、"))
	}
	if len(diagnostics.RepairPolicy) > 0 {
		lines = append(lines, "构建修复策略："+compactText(strings.Join(diagnostics.RepairPolicy, "；"), 280))
	}
	if diagnostics.RetryPolicy != "" {
		lines = append(lines, diagnostics.RetryPolicy)
	}
	return trimRoleHandoffStrings(lines, 8)
}

func workspaceBuildDiagnosticsHookNote(diagnostics *workspaceBuildDiagnostics) string {
	if diagnostics == nil {
		return "未生成构建诊断。"
	}
	if diagnostics.Status == "empty" {
		return "未提供完整构建错误，已给出读取完整错误的兜底策略。"
	}
	return fmt.Sprintf("已解析构建错误类别：%s。", firstNonEmptyString(strings.Join(diagnostics.Categories, "、"), "build_failure"))
}

func buildWorkspaceAppOperatorCapabilitySnapshot(ctx context.Context, input workspaceRoleHookInput) *workspaceAppCapabilitySnapshot {
	executeDirectory := normalizeWorkspacePath(input.ExecuteDirectory)
	snapshot := &workspaceAppCapabilitySnapshot{
		Status:           "skipped",
		ExecuteDirectory: executeDirectory,
	}
	if executeDirectory == "" {
		snapshot.Error = "execute_directory 为空，无法限定当前应用能力范围。"
		snapshot.Guidance = []string{"重新调用 change_role 时必须传目标应用目录作为 execute_directory。"}
		return snapshot
	}

	currentPath := firstNonEmptyString(input.FullCodePath, executeDirectory)
	user, app, scope := resolveSearchScopeUserApp(searchScopeCurrentApp, "", "", currentPath, searchScopeVisible)
	keyword := workspaceDirectorySearchKeyword(executeDirectory)
	snapshot.Scope = scope
	snapshot.User = user
	snapshot.App = app
	snapshot.Keyword = keyword

	resp, err := workspaceRoleHookSearchFunctions(ctx, &dto.SearchFunctionsReq{
		User:     user,
		App:      app,
		Keyword:  keyword,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		snapshot.Status = "error"
		snapshot.Error = err.Error()
		snapshot.Guidance = []string{
			"能力快照获取失败；下一步先调用 search_tools，并且 directory 必须等于 change_role.execute_directory。",
			"在拿到函数 schema 前，不要编造 full_code_path、字段名、枚举值或关联 ID。",
		}
		return snapshot
	}
	functions := []*dto.FunctionSearchResult{}
	if resp != nil {
		functions = resp.Functions
	}
	functions = filterSearchToolFunctionsByDirectory(functions, executeDirectory)
	return buildWorkspaceAppCapabilitySnapshotFromFunctions(snapshot, functions, 12)
}

func buildWorkspaceAppCapabilitySnapshotFromFunctions(snapshot *workspaceAppCapabilitySnapshot, functions []*dto.FunctionSearchResult, limit int) *workspaceAppCapabilitySnapshot {
	if snapshot == nil {
		snapshot = &workspaceAppCapabilitySnapshot{}
	}
	if limit <= 0 {
		limit = 12
	}
	snapshot.TotalFunctions = len(functions)
	snapshot.Status = "ok"
	if len(functions) == 0 {
		snapshot.Status = "empty"
		snapshot.Guidance = []string{
			"当前 execute_directory 下未发现已注册函数；先用 read_dir/search_resources 确认目录是否选错。",
			"如果目录确实没有函数，用户目标可能不是业务操作，需要重新判断是否进入产品或开发角色。",
		}
		return snapshot
	}

	for _, fn := range functions {
		if fn == nil {
			continue
		}
		switch fn.TemplateType {
		case functionschema.TypeTable:
			snapshot.Counts.Tables++
		case functionschema.TypeForm:
			snapshot.Counts.Forms++
		case functionschema.TypeChart:
			snapshot.Counts.Charts++
		}
		if len(snapshot.Functions) >= limit {
			continue
		}
		snapshot.Functions = append(snapshot.Functions, workspaceAppFunctionCapabilityFromSearchResult(fn))
	}
	snapshot.DisplayedFunctions = len(snapshot.Functions)
	snapshot.Guidance = []string{
		"用户在当前目录下的新增、提交、查询、更新、删除、查看图表，优先解释为使用现有软件完成业务结果。",
		"业务运行只能调用 execute_directory 或其子目录下函数；需要完整 schema 时继续 search_tools(directory=execute_directory, schema_output=both)。",
		"写入前必须确认必填字段、枚举、文件字段和关联 ID；必要时先查询或调用 run_on_select_fuzzy。",
	}
	return snapshot
}

func workspaceAppFunctionCapabilityFromSearchResult(fn *dto.FunctionSearchResult) workspaceAppFunctionCapability {
	summary := summarizeSearchToolSchema(fn.Schema)
	if len(summary) > 6 {
		summary = append(summary[:6], fmt.Sprintf("... 还有 %d 行字段摘要，完整 schema 用 search_tools(schema_output=both) 查看", len(summary)-6))
	}
	return workspaceAppFunctionCapability{
		Name:          compactText(fn.Name, 80),
		Code:          strings.TrimSpace(fn.Code),
		FullCodePath:  strings.TrimSpace(fn.FullCodePath),
		Type:          strings.TrimSpace(fn.TemplateType),
		Capabilities:  formatSearchToolFunctionCapabilities(fn.TemplateType, fn.Callbacks),
		RunTools:      workspaceAppRunToolsForFunction(fn),
		Description:   compactText(fn.Description, 140),
		SchemaSummary: trimRoleHandoffStrings(summary, 7),
	}
}

func workspaceAppRunToolsForFunction(fn *dto.FunctionSearchResult) []string {
	if fn == nil {
		return nil
	}
	switch fn.TemplateType {
	case functionschema.TypeForm:
		return []string{"run_form_submit"}
	case functionschema.TypeChart:
		return []string{"run_chart_query"}
	case functionschema.TypeTable:
		tools := []string{"run_table_search"}
		if hasSearchToolCallback(fn.Callbacks, "OnTableAddRow") {
			tools = append(tools, "run_table_create")
		}
		if hasSearchToolCallback(fn.Callbacks, "OnTableUpdateRow") {
			tools = append(tools, "run_table_update")
		}
		if hasSearchToolCallback(fn.Callbacks, "OnTableDeleteRows") {
			tools = append(tools, "run_table_delete")
		}
		return tools
	default:
		return nil
	}
}

func workspaceAppCapabilityHandoffLines(snapshot *workspaceAppCapabilitySnapshot) []string {
	if snapshot == nil {
		return nil
	}
	lines := []string{}
	switch snapshot.Status {
	case "ok":
		lines = append(lines, fmt.Sprintf(
			"当前应用能力快照：%s 下共 %d 个函数（Table %d / Form %d / Chart %d），已展示 %d 个；业务操作必须限定在这些函数或子目录函数内。",
			firstNonEmptyString(snapshot.ExecuteDirectory, "未指定目录"),
			snapshot.TotalFunctions,
			snapshot.Counts.Tables,
			snapshot.Counts.Forms,
			snapshot.Counts.Charts,
			snapshot.DisplayedFunctions,
		))
		for i, fn := range snapshot.Functions {
			if i >= 8 {
				break
			}
			lines = append(lines, compactText(fmt.Sprintf(
				"%s %s：%s；能力=%s；运行工具=%s；字段摘要=%s",
				firstNonEmptyString(fn.Type, "function"),
				firstNonEmptyString(fn.Name, fn.Code),
				fn.FullCodePath,
				firstNonEmptyString(fn.Capabilities, "-"),
				strings.Join(fn.RunTools, "、"),
				firstNonEmptyString(strings.Join(fn.SchemaSummary, " / "), "需用 search_tools 查看"),
			), 300))
		}
		if snapshot.TotalFunctions > 8 {
			lines = append(lines, fmt.Sprintf("当前目录还有 %d 个函数未写入交接摘要；如需选择其他函数，继续用 search_tools(directory=change_role.execute_directory) 查询。", snapshot.TotalFunctions-8))
		}
		lines = append(lines, "需要完整 schema 时调用 search_tools(directory=change_role.execute_directory, schema_output=both)；不要测试或操作整个工作区。")
	case "empty":
		lines = append(lines, "当前应用能力快照：execute_directory 下未发现已注册函数；先确认目录是否选错，不要直接当作新建系统。")
	case "error":
		lines = append(lines, "当前应用能力快照获取失败："+compactText(snapshot.Error, 180)+"；下一步先用 search_tools(directory=change_role.execute_directory) 重新确认函数 schema。")
	default:
		lines = append(lines, "当前应用能力快照未生成；重新调用 change_role 时必须明确 execute_directory。")
	}
	return trimRoleHandoffStrings(lines, 12)
}

func workspaceAppCapabilityHookNote(snapshot *workspaceAppCapabilitySnapshot) string {
	if snapshot == nil {
		return "未生成能力快照。"
	}
	switch snapshot.Status {
	case "ok":
		return fmt.Sprintf("已生成当前应用能力快照：%d 个函数。", snapshot.TotalFunctions)
	case "empty":
		return "当前目录未发现已注册函数。"
	case "error":
		return "能力快照获取失败，已给出 search_tools 兜底建议。"
	default:
		return "能力快照被跳过。"
	}
}
