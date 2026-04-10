package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestFormatSearchToolsOutputBothKeepsSummaryAndJSON(t *testing.T) {
	matchedTools := []dto.ToolDef{
		{
			Name:        "run_table_search",
			Description: "执行工作区内 Table 查询接口，返回分页表格数据。full_code_path 必须为具体表格函数路径。",
		},
	}
	functions := []*dto.FunctionSearchResult{
		{
			Name:         "工单管理",
			FullCodePath: "/liubeiluo/work/ticket_system/ticket_list.table",
			Description:  "一个简单的工单管理系统",
			TemplateType: "table",
			Request: []interface{}{
				map[string]interface{}{
					"code": "priority",
					"name": "优先级",
					"data": map[string]interface{}{
						"type": "string",
					},
					"widget": map[string]interface{}{
						"type": "select",
						"config": map[string]interface{}{
							"options":     []interface{}{"低", "中", "高"},
							"placeholder": "请选择优先级",
							"default":     "中",
						},
					},
					"validation": "required,oneof=低 中 高",
				},
				map[string]interface{}{
					"code": "handler",
					"name": "处理人",
					"data": map[string]interface{}{
						"type": "string",
					},
					"widget": map[string]interface{}{
						"type": "users",
						"config": map[string]interface{}{
							"default": "Me()",
						},
					},
				},
			},
		},
	}

	out := formatSearchToolsOutput("工单|table|查询", matchedTools, functions, searchToolsRequestOutputBoth)
	for _, want := range []string{
		"搜索结果：关键词",
		"【内置工具】",
		"run_table_search: 执行工作区内 Table 查询接口，返回分页表格数据",
		"【已注册函数摘要】",
		"字段摘要:",
		`widget=select, type=string, 【必填】, enum=低|中|高, placeholder="请选择优先级", 前端默认值=中`,
		`widget=users, type=string, format=comma-separated usernames`,
		`前端默认值=Me()`,
		`example="beiluo,zhangsan"`,
		"【已注册函数原始 request JSON】",
		`"code": "priority"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, out)
		}
	}
}

func TestSummarizeSearchToolRequestFieldsDoesNotTruncate(t *testing.T) {
	raw := make([]interface{}, 0, 13)
	for i := 1; i <= 13; i++ {
		raw = append(raw, map[string]interface{}{
			"code": fmt.Sprintf("field_%d", i),
			"data": map[string]interface{}{
				"type": "string",
			},
			"widget": map[string]interface{}{
				"type": "input",
				"config": map[string]interface{}{
					"placeholder": fmt.Sprintf("请输入字段%d", i),
				},
			},
		})
	}

	lines, err := summarizeSearchToolRequestFields(raw)
	if err != nil {
		t.Fatalf("summarizeSearchToolRequestFields returned error: %v", err)
	}
	if len(lines) != 13 {
		t.Fatalf("expected 13 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[12], `field_13: widget=input, type=string, placeholder="请输入字段13"`) {
		t.Fatalf("expected last field to be preserved, got %q", lines[12])
	}
}

func TestNormalizeSearchToolsRequestOutput(t *testing.T) {
	cases := map[string]searchToolsRequestOutput{
		"summary": searchToolsRequestOutputSummary,
		"json":    searchToolsRequestOutputJSON,
		"both":    searchToolsRequestOutputBoth,
		"":        searchToolsRequestOutputSummary,
		"weird":   searchToolsRequestOutputSummary,
	}
	for input, want := range cases {
		if got := normalizeSearchToolsRequestOutput(input); got != want {
			t.Fatalf("normalizeSearchToolsRequestOutput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatSearchToolFunctionSummaryIncludesTableCapabilities(t *testing.T) {
	fn := &dto.FunctionSearchResult{
		Name:         "支付记录",
		FullCodePath: "/liubeiluo/work/cashier/payment_record_list.table",
		TemplateType: "table",
		Callbacks:    "",
	}
	out := formatSearchToolFunctionSummary(0, fn)
	if !strings.Contains(out, "capabilities: read-only") {
		t.Fatalf("expected read-only capabilities, got:\n%s", out)
	}
}
