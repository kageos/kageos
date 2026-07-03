package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
)

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
