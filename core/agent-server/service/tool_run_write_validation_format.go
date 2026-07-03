package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/pkg/contextx"
)

func formatRunWriteValidationFailure(ctx context.Context, toolName string, issues []runWriteValidationIssue) string {
	var buf strings.Builder
	buf.WriteString(toolName)
	buf.WriteString(" 写入前校验失败，本次未提交任何数据：\n")
	grouped := groupRunWriteValidationIssues(issues)
	for _, kind := range orderedRunWriteValidationIssueKinds() {
		items := grouped[kind]
		if len(items) == 0 {
			continue
		}
		buf.WriteString("【")
		buf.WriteString(string(kind))
		buf.WriteString("】\n")
		for _, issue := range items {
			buf.WriteString("- ")
			buf.WriteString(issue.Message)
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n【给模型】\n")
	for _, kind := range orderedRunWriteValidationIssueKinds() {
		if len(grouped[kind]) == 0 {
			continue
		}
		buf.WriteString("- ")
		buf.WriteString(runWriteValidationKindGuidance(ctx, kind))
		buf.WriteString("\n")
	}
	return strings.TrimSpace(buf.String())
}

func orderedRunWriteValidationIssueKinds() []runWriteValidationIssueKind {
	return []runWriteValidationIssueKind{
		runWriteIssueRequired,
		runWriteIssueStaticChoice,
		runWriteIssueFuzzyChoice,
		runWriteIssueUser,
	}
}

func groupRunWriteValidationIssues(issues []runWriteValidationIssue) map[runWriteValidationIssueKind][]runWriteValidationIssue {
	grouped := make(map[runWriteValidationIssueKind][]runWriteValidationIssue)
	for _, issue := range issues {
		if issue.Message == "" {
			continue
		}
		grouped[issue.Kind] = append(grouped[issue.Kind], issue)
	}
	return grouped
}

func runWriteValidationKindGuidance(ctx context.Context, kind runWriteValidationIssueKind) string {
	switch kind {
	case runWriteIssueRequired:
		return "必填字段: 补齐非空值。"
	case runWriteIssueStaticChoice:
		return "静态选项: 只能填 schema options 中的值。"
	case runWriteIssueFuzzyChoice:
		return "动态选项: 用 run_on_select_fuzzy 查候选，填 items[].value；不支持 by_value/by_values 就修回调。"
	case runWriteIssueUser:
		if user := strings.TrimSpace(contextx.GetRequestUser(ctx)); user != "" {
			return "用户字段: 使用真实 username；测试时优先用当前请求用户 " + user + "。"
		}
		return "用户字段: 使用真实 username，不要填示例名或函数名。"
	default:
		return "按字段 schema 修正后重试。"
	}
}
