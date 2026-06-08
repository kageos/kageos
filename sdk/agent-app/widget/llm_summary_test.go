package widget

import (
	"strings"
	"testing"
)

func TestFieldLLMSummaryLinesIncludeTypeFormatAndExample(t *testing.T) {
	field := &Field{
		Code: "reviewers",
		Name: "审核人",
		Data: &FieldData{Type: DataTypeString},
		Widget: struct {
			Type   string      `json:"type"`
			Config interface{} `json:"config,omitempty"`
		}{
			Type:   TypeUsers,
			Config: &Users{RenderDefault: "Me()"},
		},
	}

	lines := field.LLMSummaryLines(SummaryOptions{Mode: SummaryCompact, MaxDepth: 1})
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	line := lines[0]
	for _, want := range []string{
		"widget=users",
		"type=string",
		"format=comma-separated usernames",
		"渲染默认值=Me()",
		`example="beiluo,zhangsan"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("summary line %q should contain %q", line, want)
		}
	}
}

func TestFieldLLMSummaryLinesDatetimeUsesReadableTime(t *testing.T) {
	field := &Field{
		Code: "created_at",
		Name: "创建时间",
		Data: &FieldData{Type: DataTypeString},
		Widget: struct {
			Type   string      `json:"type"`
			Config interface{} `json:"config,omitempty"`
		}{
			Type:   TypeDatetime,
			Config: &DateTime{Format: "YYYY-MM-DD HH:mm:ss"},
		},
	}

	lines := field.LLMSummaryLines(SummaryOptions{Mode: SummaryCompact, MaxDepth: 1})
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	line := lines[0]
	for _, want := range []string{
		"widget=datetime",
		"type=string",
		"format=YYYY-MM-DD HH:mm:ss",
		`example="2026-04-21 16:30:00"`,
		"storage=database datetime",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("summary line %q should contain %q", line, want)
		}
	}
}

func TestFieldLLMSummaryLinesExpandNestedChildren(t *testing.T) {
	field := &Field{
		Code:       "product_quantities",
		Name:       "商品清单",
		Validation: "required,min=1",
		Data:       &FieldData{Type: DataTypeStructs},
		Widget: struct {
			Type   string      `json:"type"`
			Config interface{} `json:"config,omitempty"`
		}{
			Type: TypeTable,
		},
		Children: []*Field{
			{
				Code:       "product_id",
				Name:       "商品",
				Validation: "required",
				Data:       &FieldData{Type: DataTypeInt},
				Widget: struct {
					Type   string      `json:"type"`
					Config interface{} `json:"config,omitempty"`
				}{
					Type:   TypeSelect,
					Config: &Select{},
				},
				Callbacks: []string{"OnSelectFuzzy"},
			},
			{
				Code:       "quantity",
				Name:       "数量",
				Validation: "required,min=1",
				Data:       &FieldData{Type: DataTypeInt},
				Widget: struct {
					Type   string      `json:"type"`
					Config interface{} `json:"config,omitempty"`
				}{
					Type: TypeInteger,
				},
			},
		},
	}

	lines := field.LLMSummaryLines(SummaryOptions{Mode: SummaryCompact, MaxDepth: 1})
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "widget=table") || !strings.Contains(lines[0], "fields=2") {
		t.Fatalf("container line missing table facts: %q", lines[0])
	}
	if !strings.Contains(lines[1], "callback=OnSelectFuzzy") ||
		!strings.Contains(lines[1], "value_source=fuzzy search result") ||
		!strings.Contains(lines[1], "候选值工具=run_on_select_fuzzy") ||
		!strings.Contains(lines[1], `调用示例=run_on_select_fuzzy(full_code_path=<当前函数路径>, code="product_id", keyword=<关键词>)`) {
		t.Fatalf("callback child line missing callback hints: %q", lines[1])
	}
	if !strings.HasPrefix(lines[1], "  - ") {
		t.Fatalf("expected nested child indentation, got %q", lines[1])
	}
}

func TestDecodeFieldsRoundTrip(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"code": "priority",
			"name": "优先级",
			"data": map[string]interface{}{
				"type": "string",
			},
			"widget": map[string]interface{}{
				"type": "select",
				"config": map[string]interface{}{
					"options":        []interface{}{"低", "中", "高"},
					"placeholder":    "请选择优先级",
					"render_default": "中",
				},
			},
			"validation": "required,oneof=低 中 高",
		},
	}

	fields, err := DecodeFields(raw)
	if err != nil {
		t.Fatalf("DecodeFields returned error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	line := fields[0].LLMSummaryLines(SummaryOptions{Mode: SummaryCompact, MaxDepth: 1})[0]
	for _, want := range []string{
		"widget=select",
		"type=string",
		"enum=低|中|高",
		`placeholder="请选择优先级"`,
		"【必填】",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("decoded summary line %q should contain %q", line, want)
		}
	}
}
