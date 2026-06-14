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
	workspaceRoleHookMaintenanceScope          = "maintenance.before_enter_scope"
	workspaceRoleHookQABeforeEnterSchema       = "qa.before_enter_schema"
)

var workspaceRoleHookSearchFunctions = apicall.SearchFunctions

const (
	workspaceRoleHookImplementationImplemented = "implemented"
	workspaceRoleHookImplementationPlanned     = "planned"
)

type workspaceRoleHookRegistration struct {
	ID        string
	Stage     string
	ShouldRun func(workspaceRoleHookInput) bool
	Run       func(context.Context, workspaceRoleHookInput) workspaceRoleHookOutput
}

var workspaceRoleHookRegistry = []workspaceRoleHookRegistration{
	{
		ID:        workspaceRoleHookProductManagerToDeveloper,
		Stage:     workspaceRoleHookStageBeforeHandoff,
		ShouldRun: shouldRunWorkspacePRDToDeveloperHook,
		Run:       runWorkspacePRDToDeveloperHook,
	},
	{
		ID:        workspaceRoleHookMaintenanceScope,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceMaintenanceScopeHook,
		Run:       runWorkspaceMaintenanceScopeHook,
	},
	{
		ID:        workspaceRoleHookQABeforeEnterSchema,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceQABeforeEnterSchemaHook,
		Run:       runWorkspaceQABeforeEnterSchemaHook,
	},
	{
		ID:        workspaceRoleHookAppOperatorCapabilities,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceAppOperatorCapabilitiesHook,
		Run:       runWorkspaceAppOperatorCapabilitiesHook,
	},
	{
		ID:        workspaceRoleHookBuildEngineerDiagnostics,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceBuildEngineerDiagnosticsHook,
		Run:       runWorkspaceBuildEngineerDiagnosticsHook,
	},
}

var workspaceRolePlannedHookIDs = map[string]struct{}{
	"product_manager.prd_ready":         {},
	"app_developer.before_enter_prd":    {},
	"app_developer.after_build":         {},
	"maintenance.after_build":           {},
	"automation.before_enter_scope":     {},
	"qa.after_run":                      {},
	"app_operator.after_run":            {},
	"build_engineer.after_build":        {},
	"data_operator.before_enter_inputs": {},
	"platform.before_enter_boundary":    {},
	"reviewer.before_handoff":           {},
}

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
	return runWorkspaceRoleHookRegistry(context.Background(), input)
}

func runWorkspaceRoleBeforeEnterHooks(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	return runWorkspaceRoleHookRegistry(ctx, input)
}

func runWorkspaceRoleHookRegistry(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	out := workspaceRoleHookOutput{}
	stage := strings.TrimSpace(input.Stage)
	for _, registration := range workspaceRoleHookRegistry {
		if strings.TrimSpace(registration.Stage) != stage || registration.Run == nil {
			continue
		}
		if registration.ShouldRun != nil && !registration.ShouldRun(input) {
			continue
		}
		mergeWorkspaceRoleHookOutput(&out, registration.Run(ctx, input))
	}
	return out
}

func mergeWorkspaceRoleHookOutput(dst *workspaceRoleHookOutput, src workspaceRoleHookOutput) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(src.PRDExecutionMarkdown) != "" {
		dst.PRDExecutionMarkdown = src.PRDExecutionMarkdown
	}
	if src.AppCapabilities != nil {
		dst.AppCapabilities = src.AppCapabilities
	}
	if src.BuildDiagnostics != nil {
		dst.BuildDiagnostics = src.BuildDiagnostics
	}
	dst.ExecutedHooks = append(dst.ExecutedHooks, src.ExecutedHooks...)
	dst.HandoffKeyInformation = append(dst.HandoffKeyInformation, src.HandoffKeyInformation...)
}

func annotateWorkspaceRoleRuntimeContractHooks(contract roleRuntimeContract) roleRuntimeContract {
	if len(contract.Hooks) == 0 {
		return contract
	}
	for i := range contract.Hooks {
		contract.Hooks[i].ImplementationStatus = workspaceRoleHookImplementationStatus(contract.Hooks[i].ID)
	}
	return contract
}

func workspaceRoleHookImplementationStatus(hookID string) string {
	hookID = strings.TrimSpace(hookID)
	if hookID == "" {
		return ""
	}
	for _, registration := range workspaceRoleHookRegistry {
		if strings.TrimSpace(registration.ID) == hookID {
			return workspaceRoleHookImplementationImplemented
		}
	}
	if _, ok := workspaceRolePlannedHookIDs[hookID]; ok {
		return workspaceRoleHookImplementationPlanned
	}
	return "unknown"
}

func runWorkspacePRDToDeveloperHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	out := workspaceRoleHookOutput{}
	sourceRole := normalizeWorkspaceRole(input.SourceRole)
	markdown := renderWorkspacePRDExecutionMarkdown(input.Artifact, input.ExecuteDirectory, input.TargetAppDirectory)
	if strings.TrimSpace(markdown) == "" {
		return out
	}
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
	return out
}

func runWorkspaceAppOperatorCapabilitiesHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	snapshot := buildWorkspaceAppOperatorCapabilitySnapshot(ctx, input)
	return workspaceRoleHookOutput{
		AppCapabilities:       snapshot,
		HandoffKeyInformation: workspaceAppCapabilityHandoffLines(snapshot),
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookAppOperatorCapabilities,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleAppOperator,
				Status:     firstNonEmptyString(snapshot.Status, "skipped"),
				Produced:   []string{"available_capabilities", "operation_schema_summary"},
				Note:       workspaceAppCapabilityHookNote(snapshot),
			},
		},
	}
}

func runWorkspaceBuildEngineerDiagnosticsHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	diagnostics := buildWorkspaceDiagnostics(workspaceBuildErrorTextFromHandoff(input.Handoff), input.ExecuteDirectory)
	return workspaceRoleHookOutput{
		BuildDiagnostics:      diagnostics,
		HandoffKeyInformation: workspaceBuildDiagnosticsHandoffLines(diagnostics),
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookBuildEngineerDiagnostics,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleBuildEngineer,
				Status:     firstNonEmptyString(diagnostics.Status, "empty"),
				Produced:   []string{"build_diagnostics", "required_docs", "repair_policy", "executed_hooks"},
				Note:       workspaceBuildDiagnosticsHookNote(diagnostics),
			},
		},
	}
}

func runWorkspaceMaintenanceScopeHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	lines := workspaceMaintenanceScopeHandoffLines(input)
	status := "ok"
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		status = "empty"
	}
	return workspaceRoleHookOutput{
		HandoffKeyInformation: lines,
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookMaintenanceScope,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleMaintenanceEngineer,
				Status:     status,
				Produced:   []string{"maintenance_scope"},
				Note:       workspaceMaintenanceScopeHookNote(input),
			},
		},
	}
}

func runWorkspaceQABeforeEnterSchemaHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	lines := workspaceQAVerificationPlanHandoffLines(input)
	status := "ok"
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		status = "empty"
	}
	return workspaceRoleHookOutput{
		HandoffKeyInformation: lines,
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookQABeforeEnterSchema,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleQAEngineer,
				Status:     status,
				Produced:   []string{"test_capability_snapshot", "verification_plan"},
				Note:       workspaceQABeforeEnterSchemaHookNote(input),
			},
		},
	}
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

func shouldRunWorkspaceMaintenanceScopeHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleMaintenanceEngineer
}

func shouldRunWorkspaceQABeforeEnterSchemaHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleQAEngineer
}

func workspaceMaintenanceScopeHandoffLines(input workspaceRoleHookInput) []string {
	executeDirectory := normalizeWorkspacePath(input.ExecuteDirectory)
	if executeDirectory == "" {
		return []string{"维护范围：execute_directory 为空；重新调用 change_role 固定目标应用目录后再读取或修改代码。"}
	}
	lines := []string{
		fmt.Sprintf("维护范围：execute_directory=%s；只读取、修改、构建该目录或其子目录，禁止扫描或改动其他应用。", executeDirectory),
	}
	paths := workspaceScopedPathsFromHandoff(input, executeDirectory)
	if len(paths) > 0 {
		lines = append(lines, "维护相关路径："+strings.Join(trimRoleHandoffStrings(paths, 8), "、"))
	} else {
		lines = append(lines, "维护相关路径：交接信息未提供具体文件；先 read_dir/read_go_file 限定 execute_directory 读取最小必要源码。")
	}
	if summary := workspaceCompactHandoffSummary(input, 220); summary != "" {
		lines = append(lines, "维护问题摘要："+summary)
	}
	lines = append(lines, "修改后只在 execute_directory 对应工作空间 build_workspace；构建/schema 问题交接 build_engineer，业务验证交接 qa_engineer。")
	return trimRoleHandoffStrings(lines, 8)
}

func workspaceQAVerificationPlanHandoffLines(input workspaceRoleHookInput) []string {
	executeDirectory := normalizeWorkspacePath(input.ExecuteDirectory)
	if executeDirectory == "" {
		return []string{"测试范围：execute_directory 为空；重新调用 change_role 固定目标应用目录后再查询 schema 或运行测试。"}
	}
	lines := []string{
		fmt.Sprintf("测试范围：execute_directory=%s；当前应用 search/run_* 调用默认围绕该目录或其子函数；需要目录内函数 schema 时调用 search(full_code_path=execute_directory, resource_type=function, schema_output=both)。", executeDirectory),
	}
	functionPaths := workspaceFunctionPaths(workspaceScopedPathsFromHandoff(input, executeDirectory))
	if len(functionPaths) > 0 {
		lines = append(lines, "候选测试函数："+strings.Join(trimRoleHandoffStrings(functionPaths, 10), "、"))
	} else {
		lines = append(lines, "候选测试函数：交接信息未提取到具体 .table/.form/.chart；先调用 search(full_code_path=change_role.execute_directory, resource_type=function, schema_output=both) 获取函数 schema。")
	}
	lines = append(lines, "验证顺序：先主数据/配置表，再 Form 提交，再目标记录表，再 Chart/结果查询；失败后归因为参数、数据、schema、业务 bug 或环境问题。")
	lines = append(lines, "测试前必须确认 Request 字段、必填项、枚举、文件字段、关联 ID 和时间/用户筛选；不要根据函数名猜 body。")
	return trimRoleHandoffStrings(lines, 8)
}

func workspaceMaintenanceScopeHookNote(input workspaceRoleHookInput) string {
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		return "execute_directory 为空，已要求重新固定维护目录。"
	}
	return "已收敛维护范围，后续源码修改和构建必须限定在 execute_directory；读取参考资料可按明确完整路径进行。"
}

func workspaceQABeforeEnterSchemaHookNote(input workspaceRoleHookInput) string {
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		return "execute_directory 为空，已要求重新固定测试目录。"
	}
	return "已生成测试范围和 schema 查询计划，后续运行工具默认围绕 execute_directory；交接中明确列出的外部函数可按完整路径调用。"
}

func workspaceScopedPathsFromHandoff(input workspaceRoleHookInput, executeDirectory string) []string {
	executeDirectory = normalizeWorkspacePath(executeDirectory)
	if executeDirectory == "" {
		return nil
	}
	text := workspaceHandoffSearchText(input)
	paths := workspacePathsFromText(text)
	out := []string{}
	for _, item := range paths {
		path := normalizeWorkspacePath(item)
		if path == "" || !workspacePathHasPrefix(path, executeDirectory) {
			continue
		}
		out = appendUniqueRoleHandoffStrings(out, path)
	}
	return out
}

func workspaceFunctionPaths(paths []string) []string {
	out := []string{}
	for _, item := range paths {
		path := normalizeWorkspacePath(item)
		if path == "" {
			continue
		}
		for _, suffix := range []string{".table", ".form", ".chart"} {
			if strings.HasSuffix(path, suffix) {
				out = appendUniqueRoleHandoffStrings(out, path)
				break
			}
		}
	}
	return out
}

func workspaceCompactHandoffSummary(input workspaceRoleHookInput, maxLength int) string {
	return compactText(strings.Join(append(append([]string{}, input.Handoff.TaskContext...), input.Handoff.KeyInformation...), "；"), maxLength)
}

func workspaceHandoffSearchText(input workspaceRoleHookInput) string {
	parts := []string{}
	parts = append(parts, input.Handoff.ExecuteDirectory)
	parts = append(parts, input.Handoff.TaskContext...)
	parts = append(parts, input.Handoff.KeyInformation...)
	parts = append(parts, input.Handoff.References...)
	return strings.Join(parts, "\n")
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

	snapshot.Keyword = executeDirectory

	resp, err := workspaceRoleHookSearchFunctions(ctx, &dto.SearchFunctionsReq{
		FullCodePath: executeDirectory,
		Page:         1,
		PageSize:     100,
	})
	if err != nil {
		snapshot.Status = "error"
		snapshot.Error = err.Error()
		snapshot.Guidance = []string{
			"能力快照获取失败；下一步先调用 search(full_code_path=change_role.execute_directory, resource_type=function, schema_output=both)。",
			"在拿到函数 schema 前，不要编造 full_code_path、字段名、枚举值或关联 ID。",
		}
		return snapshot
	}
	functions := []*dto.FunctionSearchResult{}
	if resp != nil {
		functions = resp.Functions
	}
	functions = filterSearchFunctionsByFullCodePath(functions, executeDirectory)
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
			"当前 execute_directory 下未发现已注册函数；先用 read_dir/search 确认目录是否选错。",
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
		"业务运行只能调用 execute_directory 或其子目录下函数；当前应用完整 schema 用 search(full_code_path=execute_directory, resource_type=function, schema_output=both)。",
		"写入前必须确认必填字段、枚举、文件字段和关联 ID；必要时先查询或调用 run_on_select_fuzzy。",
	}
	return snapshot
}

func workspaceAppFunctionCapabilityFromSearchResult(fn *dto.FunctionSearchResult) workspaceAppFunctionCapability {
	summary := summarizeSearchSchema(fn.Schema)
	if len(summary) > 6 {
		summary = append(summary[:6], fmt.Sprintf("... 还有 %d 行字段摘要，完整 schema 用 search(schema_output=both) 查看", len(summary)-6))
	}
	return workspaceAppFunctionCapability{
		Name:          compactText(fn.Name, 80),
		Code:          strings.TrimSpace(fn.Code),
		FullCodePath:  strings.TrimSpace(fn.FullCodePath),
		Type:          strings.TrimSpace(fn.TemplateType),
		Capabilities:  formatSearchFunctionCapabilities(fn.TemplateType, fn.Callbacks),
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
		if hasSearchCallback(fn.Callbacks, "OnTableAddRow") {
			tools = append(tools, "run_table_create")
		}
		if hasSearchCallback(fn.Callbacks, "OnTableUpdateRow") {
			tools = append(tools, "run_table_update")
		}
		if hasSearchCallback(fn.Callbacks, "OnTableDeleteRows") {
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
			"当前应用能力快照：%s 下共 %d 个函数（Table %d / Form %d / Chart %d），已展示 %d 个；业务操作默认优先使用这些函数或子目录函数，若用户或 SOP 明确给出外部函数完整路径，可按完整路径调用并由平台权限判断。",
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
				firstNonEmptyString(strings.Join(fn.SchemaSummary, " / "), "需用 search 查看"),
			), 300))
		}
		if snapshot.TotalFunctions > 8 {
			lines = append(lines, fmt.Sprintf("当前目录还有 %d 个函数未写入交接摘要；如需选择其他函数，继续用 search(full_code_path=change_role.execute_directory, resource_type=function) 查询。", snapshot.TotalFunctions-8))
		}
		lines = append(lines, "需要完整 schema 时调用 search(full_code_path=change_role.execute_directory, resource_type=function, schema_output=both)。")
	case "empty":
		lines = append(lines, "当前应用能力快照：execute_directory 下未发现已注册函数；先确认目录是否选错，不要直接当作新建系统。")
	case "error":
		lines = append(lines, "当前应用能力快照获取失败："+compactText(snapshot.Error, 180)+"；下一步先用 search(full_code_path=change_role.execute_directory, resource_type=function) 重新确认函数 schema。")
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
		return "能力快照获取失败，已给出 search 兜底建议。"
	default:
		return "能力快照被跳过。"
	}
}
