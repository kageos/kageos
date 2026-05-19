package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
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
			Schema: functionschema.NewTable(
				[]*widget.Field{
					testSearchToolField("priority", "优先级", "select", map[string]interface{}{
						"options":        []interface{}{"低", "中", "高"},
						"placeholder":    "请选择优先级",
						"render_default": "中",
					}, "required,oneof=低 中 高"),
					testSearchToolField("handler", "处理人", "users", map[string]interface{}{
						"render_default": "Me()",
					}, ""),
				},
				nil,
				nil,
			),
		},
	}

	out := formatSearchToolsOutput("工单|table|查询", matchedTools, functions, searchToolsRequestOutputBoth)
	for _, want := range []string{
		"搜索结果：关键词",
		"【内置工具】",
		"run_table_search: 执行工作区内 Table 查询接口，返回分页表格数据",
		"【已注册函数摘要】",
		"字段摘要:",
		`widget=select, type=string, 【必填】, enum=低|中|高, placeholder="请选择优先级", 渲染默认值=中`,
		`widget=users, type=string, format=comma-separated usernames`,
		`渲染默认值=Me()`,
		`example="beiluo,zhangsan"`,
		"【已注册函数 Schema JSON】",
		`"code": "priority"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, out)
		}
	}
}

func TestSummarizeSearchToolSchemaDoesNotTruncate(t *testing.T) {
	fields := make([]*widget.Field, 0, 13)
	for i := 1; i <= 13; i++ {
		fields = append(fields, testSearchToolField(
			fmt.Sprintf("field_%d", i),
			"",
			"input",
			map[string]interface{}{"placeholder": fmt.Sprintf("请输入字段%d", i)},
			"",
		))
	}

	lines := summarizeSearchToolSchema(functionschema.NewForm(fields, nil, nil))
	if len(lines) != 14 {
		t.Fatalf("expected 13 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[13], `field_13: widget=input, type=string, placeholder="请输入字段13"`) {
		t.Fatalf("expected last field to be preserved, got %q", lines[13])
	}
}

func TestNormalizeSearchToolsRequestOutput(t *testing.T) {
	cases := map[string]searchToolsRequestOutput{
		"summary": searchToolsRequestOutputSummary,
		"both":    searchToolsRequestOutputBoth,
		"json":    searchToolsRequestOutputSummary,
		"":        searchToolsRequestOutputSummary,
		"weird":   searchToolsRequestOutputSummary,
	}
	for input, want := range cases {
		if got := normalizeSearchToolsRequestOutput(input); got != want {
			t.Fatalf("normalizeSearchToolsRequestOutput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveSearchScopeUserApp(t *testing.T) {
	user, app, scope := resolveSearchScopeUserApp("current_app", "", "", "/beiluo/demo/pkg/ticket_list.table", searchScopeSystem)
	if user != "beiluo" || app != "demo" || scope != searchScopeCurrentApp {
		t.Fatalf("unexpected current_app scope: user=%q app=%q scope=%q", user, app, scope)
	}

	user, app, scope = resolveSearchScopeUserApp("visible", "system", "tools", "/beiluo/demo", searchScopeSystem)
	if user != "system" || app != "tools" || scope != searchScopeVisible {
		t.Fatalf("explicit user/app should win: user=%q app=%q scope=%q", user, app, scope)
	}
}

func TestFilterSearchToolFunctionsByCapability(t *testing.T) {
	functions := []*dto.FunctionSearchResult{
		{Name: "只读表", TemplateType: "table"},
		{Name: "可编辑表", TemplateType: "table", Callbacks: []string{"OnTableUpdateRow"}},
		{Name: "表单", TemplateType: "form"},
	}

	got := filterSearchToolFunctionsByCapability(functions, "update")
	if len(got) != 1 || got[0].Name != "可编辑表" {
		t.Fatalf("expected only editable table, got %#v", got)
	}

	got = filterSearchToolFunctionsByCapability(functions, "read-only")
	if len(got) != 1 || got[0].Name != "只读表" {
		t.Fatalf("expected only read-only table, got %#v", got)
	}

	got = filterSearchToolFunctionsByCapability(functions, "submit")
	if len(got) != 1 || got[0].Name != "表单" {
		t.Fatalf("expected only form, got %#v", got)
	}
}

func TestPaginateSearchToolFunctions(t *testing.T) {
	functions := []*dto.FunctionSearchResult{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
	}
	got := paginateSearchToolFunctions(functions, 2, 2)
	if len(got) != 1 || got[0].Name != "C" {
		t.Fatalf("unexpected paginated functions: %#v", got)
	}
	if got := paginateSearchToolFunctions(functions, 3, 2); len(got) != 0 {
		t.Fatalf("expected empty out-of-range page, got %#v", got)
	}
}

func TestFormatSearchToolFunctionSummaryIncludesTableCapabilities(t *testing.T) {
	fn := &dto.FunctionSearchResult{
		Name:         "支付记录",
		FullCodePath: "/liubeiluo/work/cashier/payment_record_list.table",
		TemplateType: "table",
		Callbacks:    nil,
	}
	out := formatSearchToolFunctionSummary(0, fn)
	if !strings.Contains(out, "capabilities: read-only") {
		t.Fatalf("expected read-only capabilities, got:\n%s", out)
	}

	fn.Callbacks = []string{"OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRows"}
	out = formatSearchToolFunctionSummary(0, fn)
	if !strings.Contains(out, "capabilities: read, create, update, delete") {
		t.Fatalf("expected executable table capabilities, got:\n%s", out)
	}
}

func testSearchToolField(code, name, widgetType string, config map[string]interface{}, validation string) *widget.Field {
	return &widget.Field{
		Code:       code,
		Name:       name,
		Data:       &widget.FieldData{Type: "string"},
		Validation: validation,
		Widget: struct {
			Type   string      `json:"type"`
			Config interface{} `json:"config,omitempty"`
		}{
			Type:   widgetType,
			Config: config,
		},
	}
}
