package service

import (
	"fmt"
	"path/filepath"
	"strings"
)

const maxInlineGoDiagnostics = 12

func appendGoFileDiagnostics(message string, directory string, fileName string, source string) string {
	if !isGoFileName(fileName) || strings.TrimSpace(source) == "" {
		return message
	}
	result := checkGoFileLocalSource(directory, goSourceFileForCheck{
		Name:    fileName,
		Content: source,
	})
	if result.IssueCount == 0 {
		return message + "\n自动代码诊断: 未发现当前文件的常见 Go/SDK/schema 问题。"
	}
	var b strings.Builder
	b.WriteString(message)
	b.WriteString("\n自动代码诊断: 发现 ")
	b.WriteString(fmt.Sprintf("%d", result.IssueCount))
	b.WriteString(" 个文件级非阻断问题，文件已落盘。若当前任务还在创建/重写多个文件，不要中断后续文件写入；先继续写完本轮计划文件，完整落盘后再批量修复这些诊断并统一 build。此诊断不检查跨文件依赖，跨文件/schema 问题以最终 build 为准。")
	limit := result.IssueCount
	if limit > maxInlineGoDiagnostics {
		limit = maxInlineGoDiagnostics
	}
	for i := 0; i < limit; i++ {
		issue := result.Issues[i]
		b.WriteString("\n- [")
		b.WriteString(issue.Severity)
		b.WriteString("] ")
		b.WriteString(issue.Category)
		if issue.Line > 0 {
			b.WriteString(fmt.Sprintf(":%d", issue.Line))
		}
		b.WriteString(" - ")
		b.WriteString(issue.Message)
	}
	if result.IssueCount > limit {
		b.WriteString(fmt.Sprintf("\n- 其余 %d 个文件级问题未展开；请在本轮文件完整落盘后一起处理。", result.IssueCount-limit))
	}
	return b.String()
}

func isGoFileName(fileName string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(fileName)), ".go")
}
