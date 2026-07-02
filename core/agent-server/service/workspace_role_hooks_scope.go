package service

import (
	"fmt"
	"strings"
)

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
