package prd

import (
	"strings"
	"testing"
)

func TestContractMarkdownDocumentsV2Shape(t *testing.T) {
	got := ContractMarkdown()
	for _, want := range []string{
		"## PRD 规则",
		"`project/tables/forms/charts/workflow/rules`",
		"`search_fields` 只描述搜索参数",
		"`创建开始时间`、`创建结束时间`",
		"## 代表性输出示例",
		`"workflow"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("contract markdown should contain %q, got:\n%s", want, got)
		}
	}
}

func TestApplyContractMarkdown(t *testing.T) {
	got := ApplyContractMarkdown("before\n" + ContractMarker + "\nafter")
	if strings.Contains(got, ContractMarker) {
		t.Fatalf("contract marker should be replaced: %s", got)
	}
	if !strings.Contains(got, "`workflow` 是用户展示和操作顺序") {
		t.Fatalf("contract markdown not injected: %s", got)
	}
}

func TestSupportedPRDContractValues(t *testing.T) {
	if !IsSupportedWidget("text_area") || !IsSupportedWidget("TEXTAREA") {
		t.Fatalf("text area aliases should be supported")
	}
	if IsSupportedWidget("name:状态;type:select") {
		t.Fatalf("widget tag should not be supported")
	}
	if !IsSupportedHandler("OnTableAddRow") || IsSupportedHandler("OnTableReadonly") {
		t.Fatalf("handler whitelist mismatch")
	}
	if !IsSupportedChartType("line") || IsSupportedChartType("gauge") {
		t.Fatalf("chart type whitelist mismatch")
	}
	if got := NormalizeChartDimension("日期（按天/周/月）"); got != "日期" {
		t.Fatalf("NormalizeChartDimension = %q, want 日期", got)
	}
}

func TestAllowedKeySetsExposeV2Only(t *testing.T) {
	top := AllowedTopLevelKeys()
	for _, want := range []string{"project", "tables", "forms", "charts", "workflow", "rules"} {
		if _, ok := top[want]; !ok {
			t.Fatalf("top-level keys should contain %q", want)
		}
	}
	for _, legacy := range []string{"models", "functions", "features", "acceptance_cases", "confirmation"} {
		if _, ok := top[legacy]; ok {
			t.Fatalf("top-level keys should not contain legacy key %q", legacy)
		}
	}
}
