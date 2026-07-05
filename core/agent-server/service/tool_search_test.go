package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
)

func TestFormatSearchOutputBothKeepsSummaryAndJSON(t *testing.T) {
	data := searchResultData{
		Keyword:  "工单|table|查询",
		Page:     1,
		PageSize: 20,
		MatchedTools: []dto.ToolDef{
			{
				Name:        "run_table_search",
				Description: "执行工作区内 Table 查询接口，返回分页表格数据。full_code_path 必须为具体表格函数路径。",
			},
		},
		Functions: []*dto.FunctionSearchResult{
			{
				Name:         "工单管理",
				FullCodePath: "/liubeiluo/work/ticket_system/ticket_list.table",
				Description:  "一个简单的工单管理系统",
				TemplateType: "table",
				Schema: functionschema.NewTable(
					[]*widget.Field{
						testSearchField("priority", "优先级", "select", map[string]interface{}{
							"options":        []interface{}{"低", "中", "高"},
							"placeholder":    "请选择优先级",
							"render_default": "中",
						}, "required,oneof=低 中 高"),
						testSearchField("handler", "处理人", "users", map[string]interface{}{
							"render_default": "Me()",
						}, ""),
					},
					nil,
					nil,
				),
			},
		},
	}
	data.Total = int64(len(data.MatchedTools) + len(data.Functions))

	out := formatSearchOutput(data, searchRequestOutputBoth)
	for _, want := range []string{
		"搜索结果：工单|table|查询",
		"【内置工具】",
		"run_table_search: 执行工作区内 Table 查询接口，返回分页表格数据",
		"token: <tool:run_table_search>",
		"【可执行函数】",
		"字段摘要:",
		`widget=select, type=string, 【必填】, enum=低|中|高, placeholder="请选择优先级", 渲染默认值=中`,
		`widget=users, type=string, format=comma-separated usernames`,
		`渲染默认值=Me()`,
		`example="beiluo,zhangsan"`,
		"【函数 Schema JSON】",
		`"code": "priority"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, out)
		}
	}
}

func TestSummarizeSearchSchemaDoesNotTruncate(t *testing.T) {
	fields := make([]*widget.Field, 0, 13)
	for i := 1; i <= 13; i++ {
		fields = append(fields, testSearchField(
			fmt.Sprintf("field_%d", i),
			"",
			"input",
			map[string]interface{}{"placeholder": fmt.Sprintf("请输入字段%d", i)},
			"",
		))
	}

	lines := summarizeSearchSchema(functionschema.NewForm(fields, nil, nil))
	if len(lines) != 14 {
		t.Fatalf("expected 13 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[13], `field_13: widget=input, type=string, placeholder="请输入字段13"`) {
		t.Fatalf("expected last field to be preserved, got %q", lines[13])
	}
}

func TestSearchSchemaAcceptsFullCodePathAndHasNoScope(t *testing.T) {
	schema := (&SearchTool{}).Definition().InputSchema
	err := validateToolArguments(schema, map[string]interface{}{
		"full_code_path": "/system/demos/weixin/wechat_articles/search_articles.form",
		"resource_type":  "function",
		"schema_output":  "both",
	})
	if err != nil {
		t.Fatalf("full_code_path should be accepted by search schema: %v", err)
	}
	properties, _ := schema["properties"].(map[string]interface{})
	for _, oldField := range []string{"scope", "directory", "user", "app"} {
		if _, ok := properties[oldField]; ok {
			t.Fatalf("search schema should not expose %s: %#v", oldField, schema)
		}
	}
}

func TestSearchPromptDocPathMatchesLooseContentTokens(t *testing.T) {
	result := runSearchTool(context.Background(), nil, searchArgs{
		Keyword:      "callback OnSelectFuzzy",
		FullCodePath: "/system/prompt/case_catalog/tables/hr",
		ResourceType: "tool",
	})
	if result.IsError {
		t.Fatalf("search should not error: %#v", result)
	}
	for _, want := range []string{
		"【内置文档内容命中】",
		"/system/prompt/case_catalog/tables/hr",
		`callback:"OnSelectFuzzy"`,
		"read_doc(directory=<full_code_path>)",
		"不要用 read_file",
	} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("search output should contain %q, got:\n%s", want, result.Content)
		}
	}
}

func TestSearchPromptDocPseudoFilePathFallsBackToCaseDoc(t *testing.T) {
	result := runSearchTool(context.Background(), nil, searchArgs{
		Keyword:      "callback OnSelectFuzzy",
		FullCodePath: "/system/prompt/case_catalog/tables/hr/hr_resume_list.go",
		ResourceType: "tool",
	})
	if result.IsError {
		t.Fatalf("search should not error: %#v", result)
	}
	for _, want := range []string{
		"【内置文档内容命中】",
		"full_code_path: /system/prompt/case_catalog/tables/hr",
		`callback:"OnSelectFuzzy"`,
	} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("search output should contain %q, got:\n%s", want, result.Content)
		}
	}
}

func TestSearchPromptDocPathNoMatchSuggestsReadDoc(t *testing.T) {
	result := runSearchTool(context.Background(), nil, searchArgs{
		Keyword:      "definitely-not-in-hr-case",
		FullCodePath: "/system/prompt/case_catalog/tables/hr",
		ResourceType: "tool",
	})
	if result.IsError {
		t.Fatalf("search should not error: %#v", result)
	}
	for _, want := range []string{
		"未在内置文档/案例 /system/prompt/case_catalog/tables/hr 中命中内容",
		`read_doc(directory="/system/prompt/case_catalog/tables/hr")`,
		"不要用 read_file",
	} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("search miss output should contain %q, got:\n%s", want, result.Content)
		}
	}
}

func TestNormalizeSearchResourceType(t *testing.T) {
	cases := map[string]string{
		"":          "all",
		"all":       "all",
		"doc":       "docs",
		"document":  "docs",
		"function":  "function",
		"package":   "directory",
		"directory": "directory",
		"tool":      "tool",
		"weird":     "all",
	}
	for input, want := range cases {
		if got := normalizeSearchResourceType(input); got != want {
			t.Fatalf("normalizeSearchResourceType(%q)=%q want %q", input, got, want)
		}
	}
}

func TestNormalizeSearchRequestOutput(t *testing.T) {
	cases := map[string]searchRequestOutput{
		"summary": searchRequestOutputSummary,
		"both":    searchRequestOutputBoth,
		"json":    searchRequestOutputSummary,
		"":        searchRequestOutputSummary,
		"weird":   searchRequestOutputSummary,
	}
	for input, want := range cases {
		if got := normalizeSearchRequestOutput(input); got != want {
			t.Fatalf("normalizeSearchRequestOutput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFilterSearchFunctionsByCapability(t *testing.T) {
	functions := []*dto.FunctionSearchResult{
		{Name: "只读表", TemplateType: "table"},
		{Name: "可编辑表", TemplateType: "table", Callbacks: []string{"OnTableUpdateRow"}},
		{Name: "表单", TemplateType: "form"},
	}

	got := filterSearchFunctionsByCapability(functions, "update")
	if len(got) != 1 || got[0].Name != "可编辑表" {
		t.Fatalf("expected only editable table, got %#v", got)
	}

	got = filterSearchFunctionsByCapability(functions, "read-only")
	if len(got) != 1 || got[0].Name != "只读表" {
		t.Fatalf("expected only read-only table, got %#v", got)
	}

	got = filterSearchFunctionsByCapability(functions, "submit")
	if len(got) != 1 || got[0].Name != "表单" {
		t.Fatalf("expected only form, got %#v", got)
	}
}

func TestPaginateSearchFunctions(t *testing.T) {
	functions := []*dto.FunctionSearchResult{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
	}
	got := paginateSearchFunctions(functions, 2, 2)
	if len(got) != 1 || got[0].Name != "C" {
		t.Fatalf("unexpected paginated functions: %#v", got)
	}
	if got := paginateSearchFunctions(functions, 3, 2); len(got) != 0 {
		t.Fatalf("expected empty out-of-range page, got %#v", got)
	}
}

func TestFormatSearchFunctionSummaryIncludesTableCapabilities(t *testing.T) {
	fn := &dto.FunctionSearchResult{
		Name:         "支付记录",
		FullCodePath: "/liubeiluo/work/cashier/payment_record_list.table",
		TemplateType: "table",
		Callbacks:    nil,
	}
	out := formatSearchFunctionSummary(0, fn)
	if !strings.Contains(out, "capabilities: read-only") {
		t.Fatalf("expected read-only capabilities, got:\n%s", out)
	}

	fn.Callbacks = []string{"OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRows"}
	out = formatSearchFunctionSummary(0, fn)
	if !strings.Contains(out, "capabilities: read, create, update, delete") {
		t.Fatalf("expected executable table capabilities, got:\n%s", out)
	}
}

func TestFormatSearchResourceSummaryIncludesMetadata(t *testing.T) {
	out := formatSearchResourceSummary(0, &dto.ResourceSearchResult{
		Name:         "工单列表",
		Type:         "function",
		TemplateType: "table",
		FullCodePath: "/beiluo/demo/tickets/ticket_list.table",
		AppUser:      "beiluo",
		AppCode:      "demo",
		MatchSource:  "node",
		Description:  "查询工单",
		RunCount:     3,
	})

	for _, want := range []string{
		"工单列表",
		"full_code_path: /beiluo/demo/tickets/ticket_list.table",
		"type: function / table",
		"app: beiluo/demo",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, out)
		}
	}
}

func TestFilterSearchResourceItemsByType(t *testing.T) {
	items := []*dto.ResourceSearchResult{
		{Name: "工单目录", Type: "package"},
		{Name: "工单列表", Type: "function"},
		{Name: "工单文档", Type: "docs"},
	}
	got := filterSearchResourceItemsByType(items, "function", false)
	if len(got) != 2 || got[0].Type == "function" || got[1].Type == "function" {
		t.Fatalf("expected functions to be excluded, got %#v", got)
	}
}

func testSearchField(code, name, widgetType string, config map[string]interface{}, validation string) *widget.Field {
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
