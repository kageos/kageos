package service

import (
	"fmt"
	"strings"
)

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
