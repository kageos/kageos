package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWritePRDToolReturnsStructuredPreview(t *testing.T) {
	reg := NewToolRegistry(nil)
	result := reg.CallTool(context.Background(), "write_prd", map[string]interface{}{
		"project": map[string]interface{}{
			"name":                 "会员收银系统",
			"code":                 "cashier",
			"create_new_directory": true,
			"summary":              "管理会员收银和支付记录",
			"reason":               "收银和支付记录是独立业务域",
		},
		"models": []map[string]interface{}{
			{
				"name": "支付记录",
				"fields": []map[string]interface{}{
					{
						"name":   "订单号",
						"widget": "name:订单号;type:input",
						"hide":   "create,update",
					},
					{
						"name":     "状态",
						"widget":   "name:状态;type:select;options:支付成功,支付失败;options_colors:67C23A,F56C6C;render_default:支付成功",
						"validate": "required",
					},
				},
			},
		},
		"functions": []map[string]interface{}{
			{
				"title":       "支付记录表",
				"type":        "table",
				"route":       "cashier_payment_record_list.table",
				"model":       "支付记录",
				"description": "仅列表查询，记录由收银台 Form 产生",
				"table": map[string]interface{}{
					"capability": "仅列表查询",
					"readonly":   true,
					"request_fields": []map[string]interface{}{
						{
							"field":       "状态",
							"type":        "下拉选择",
							"required":    false,
							"default":     "全部",
							"description": "按支付状态筛选",
						},
						{
							"field":       "创建时间",
							"type":        "日期时间",
							"required":    false,
							"default":     "—",
							"description": "按创建时间范围搜索",
						},
					},
					"columns": []string{"创建时间", "订单号", "实付金额", "状态"},
					"sample_rows": []map[string]interface{}{
						{
							"创建时间": "2025-01-20 11:30",
							"订单号":  "ORD202501200001",
							"实付金额": 13.50,
							"状态":   "支付成功",
						},
					},
				},
			},
			{
				"title": "收银台 Form",
				"type":  "form",
				"route": "cashier_desk.form",
				"model": "支付记录",
				"form": map[string]interface{}{
					"request_fields": []map[string]interface{}{
						{
							"field":       "商品清单",
							"type":        "表格",
							"required":    true,
							"default":     "—",
							"description": "type:table，至少 1 行",
						},
					},
					"response_fields": []map[string]interface{}{
						{
							"field":       "订单号",
							"type":        "文本",
							"example":     "ORD202501200001",
							"description": "支付成功后返回订单号",
						},
					},
				},
			},
			{
				"title": "支付金额趋势",
				"type":  "chart",
				"route": "cashier_payment_trend.chart",
				"model": "支付记录",
				"chart": map[string]interface{}{
					"chart_type": "LineChart",
					"dimension":  "日期",
					"metrics":    []string{"实付金额"},
					"request_fields": []map[string]interface{}{
						{
							"field":       "时间范围",
							"type":        "日期时间",
							"required":    false,
							"default":     "近 7 天",
							"description": "统计时间范围",
						},
					},
					"response_fields": []map[string]interface{}{
						{
							"field":       "总实付金额",
							"type":        "数字",
							"example":     "1280.50",
							"description": "放入图表 Metadata",
						},
					},
				},
			},
		},
		"acceptance_cases": []map[string]interface{}{
			{"name": "查询列表", "action": "打开支付记录表", "expected": "看到支付记录"},
		},
	}, "/liubeiluo/ccc", "")
	if result.IsError {
		t.Fatalf("write_prd returned error: %s", result.Content)
	}
	data, ok := result.Data.(writePRDResultData)
	if !ok {
		t.Fatalf("write_prd data type = %T, want writePRDResultData", result.Data)
	}
	if data.Kind != "agent_app_prd" {
		t.Fatalf("write_prd kind = %q", data.Kind)
	}
	if data.Project.Code != "cashier" || !data.Project.CreateNewDirectory {
		t.Fatalf("unexpected project: %#v", data.Project)
	}
	if len(data.Models) != 1 || data.Models[0].Name != "支付记录" {
		t.Fatalf("unexpected models: %#v", data.Models)
	}
	if len(data.Functions) != 3 {
		t.Fatalf("functions len = %d", len(data.Functions))
	}
	for idx, fn := range data.Functions {
		if fn.Order != idx+1 {
			t.Fatalf("function order = %d, want %d for %#v", fn.Order, idx+1, fn)
		}
	}
	if got := len(data.Functions[0].Table.RequestFields); got != 2 {
		t.Fatalf("table request_fields len = %d, want 2", got)
	}
	if got := data.Functions[0].Table.SampleRows[0]["实付金额"]; got != 13.50 {
		t.Fatalf("sample row amount = %#v, want numeric 13.50", got)
	}
	if data.Functions[1].Method != "POST" {
		t.Fatalf("form method default = %q, want POST", data.Functions[1].Method)
	}
	if got := len(data.Functions[1].Form.ResponseFields); got != 1 {
		t.Fatalf("form response_fields len = %d, want 1", got)
	}
	if got := len(data.Functions[2].Chart.RequestFields); got != 1 {
		t.Fatalf("chart request_fields len = %d, want 1", got)
	}
	if got := len(data.Functions[2].Chart.ResponseFields); got != 1 {
		t.Fatalf("chart response_fields len = %d, want 1", got)
	}
	if data.Confirmation.Question == "" {
		t.Fatal("confirmation question should be defaulted")
	}
	if !strings.Contains(result.Content, "PRD 预览已生成") {
		t.Fatalf("result content should contain preview notice, got %q", result.Content)
	}
}

func TestWritePRDToolAcceptsLightweightChartPreview(t *testing.T) {
	reg := NewToolRegistry(nil)
	result := reg.CallTool(context.Background(), "write_prd", map[string]interface{}{
		"project": map[string]interface{}{
			"name":                 "工单管理",
			"code":                 "ticket",
			"create_new_directory": true,
			"reason":               "独立管理工单",
		},
		"models": []map[string]interface{}{
			{
				"name": "工单",
				"fields": []map[string]interface{}{
					{"name": "标题", "widget": "name:标题;type:input", "validate": "required"},
					{"name": "状态", "widget": "name:状态;type:select;options:待处理,处理中,已完成", "validate": "required"},
				},
			},
		},
		"functions": []map[string]interface{}{
			{
				"title": "工单趋势",
				"type":  "chart",
				"route": "ticket_trend.chart",
				"model": "工单",
				"chart": map[string]interface{}{
					"chart_type": "line",
					"dimension":  "日期",
					"metrics":    []string{"新增工单", "完成工单"},
					"filters": []map[string]interface{}{
						{"name": "开始时间", "type": "datetime", "required": false, "desc": "统计开始时间"},
						{"name": "结束时间", "type": "datetime", "required": false, "desc": "统计结束时间"},
					},
					"preview_data": []map[string]interface{}{
						{"日期": "2025-01-18", "新增工单": 8, "完成工单": 5},
						{"日期": "2025-01-19", "新增工单": 10, "完成工单": 7},
					},
					"summary": []map[string]interface{}{
						{"name": "总工单数", "value": 18, "desc": "当前筛选范围内的新增工单数"},
					},
				},
			},
			{
				"title": "工单分类",
				"type":  "chart",
				"route": "ticket_category.chart",
				"model": "工单",
				"chart": map[string]interface{}{
					"chart_type":   "bar",
					"dimension":    "分类",
					"metrics":      []string{"工单数量"},
					"preview_data": []map[string]interface{}{{"分类": "设备", "工单数量": 12}},
				},
			},
			{
				"title": "状态分布",
				"type":  "chart",
				"route": "ticket_status.chart",
				"model": "工单",
				"chart": map[string]interface{}{
					"chart_type":   "pie",
					"dimension":    "状态",
					"metrics":      []string{"工单数量"},
					"preview_data": []map[string]interface{}{{"name": "待处理", "value": 6}},
				},
			},
		},
	}, "/liubeiluo/ccc", "")
	if result.IsError {
		t.Fatalf("write_prd should accept lightweight chart preview, got: %s", result.Content)
	}
	data, ok := result.Data.(writePRDResultData)
	if !ok {
		t.Fatalf("write_prd data type = %T, want writePRDResultData", result.Data)
	}
	if got := len(data.Functions); got != 3 {
		t.Fatalf("functions len = %d, want 3", got)
	}
	if got := len(data.Functions[0].Chart.Filters); got != 2 {
		t.Fatalf("chart filters len = %d, want 2", got)
	}
	if got := data.Functions[0].Chart.Filters[0].Type; got != "datetime" {
		t.Fatalf("chart filter type = %q, want datetime", got)
	}
	if got := data.Functions[0].Chart.PreviewData[0]["新增工单"]; got != float64(8) {
		t.Fatalf("preview data value = %#v, want 8", got)
	}
	if got := data.Functions[2].Chart.ChartType; got != "pie" {
		t.Fatalf("third chart type = %q, want pie", got)
	}
}

func TestWritePRDToolAcceptsJSONEncodedFunctions(t *testing.T) {
	reg := NewToolRegistry(nil)
	functions := `[
		{
			"title": "工单列表",
			"type": "table",
			"route": "ticket_list.table",
			"model": "工单",
			"table": {
				"request_fields": [],
				"columns": ["标题", "状态"],
				"sample_rows": [{"标题": "打印机无法连接", "状态": "待处理"}]
			}
		}
	]`
	result := reg.CallTool(context.Background(), "write_prd", map[string]interface{}{
		"project": map[string]interface{}{
			"name":                 "工单管理",
			"code":                 "ticket",
			"create_new_directory": true,
			"reason":               "独立管理工单",
		},
		"models": []map[string]interface{}{
			{
				"name": "工单",
				"fields": []map[string]interface{}{
					{"name": "标题", "widget": "name:标题;type:input", "validate": "required"},
					{"name": "状态", "widget": "name:状态;type:select;options:待处理,处理中,已完成", "validate": "required"},
				},
			},
		},
		"functions": functions,
	}, "/liubeiluo/ccc", "")
	if result.IsError {
		t.Fatalf("write_prd should accept JSON encoded functions, got: %s", result.Content)
	}
}

func TestWritePRDArgsDecodeAcceptsJSONStringFields(t *testing.T) {
	functions := `[
		{
			"title": "工单状态统计",
			"type": "chart",
			"route": "ticket_stats.chart",
			"model": "工单",
			"chart": {
				"chart_type": "pie",
				"dimension": "工单状态",
				"metrics": ["工单数量"],
				"filters": [
					{"name": "开始时间", "type": "datetime", "required": false, "desc": "创建时间开始"},
					{"name": "结束时间", "type": "datetime", "required": false, "desc": "创建时间结束"}
				],
				"preview_data": [
					{"name": "待处理", "value": 15},
					{"name": "处理中", "value": 8},
					{"name": "已完成", "value": 42}
				]
			}
		}
	]`
	args, err := decodeToolArgs[writePRDArgs](map[string]interface{}{
		"project":          `{"name":"工单管理","code":"ticket","create_new_directory":true,"reason":"独立管理工单"}`,
		"models":           `[{"name":"工单","fields":[{"name":"工单标题","widget":"name:工单标题;type:input","validate":"required"}]}]`,
		"functions":        functions,
		"acceptance_cases": `[{"name":"查看统计","action":"进入图表","expected":"看到状态分布"}]`,
		"confirmation":     `{"question":"确认后开始创建"}`,
	})
	if err != nil {
		t.Fatalf("decodeToolArgs should accept JSON string fields: %v", err)
	}
	if args.Project.Code != "ticket" {
		t.Fatalf("project code = %q, want ticket", args.Project.Code)
	}
	if len(args.Models) != 1 || args.Models[0].Name != "工单" {
		t.Fatalf("unexpected models: %#v", args.Models)
	}
	if len(args.Functions) != 1 || args.Functions[0].Chart == nil {
		t.Fatalf("unexpected functions: %#v", args.Functions)
	}
	if got := args.Functions[0].Chart.PreviewData[0]["value"]; got != float64(15) {
		t.Fatalf("preview value = %#v, want 15", got)
	}
	if len(args.AcceptanceCases) != 1 || args.Confirmation.Question == "" {
		t.Fatalf("unexpected acceptance/confirmation: %#v %#v", args.AcceptanceCases, args.Confirmation)
	}
}

func TestWritePRDToolRepairsJSONStringFunctionsWithPrematureObjectClose(t *testing.T) {
	reg := NewToolRegistry(nil)
	functions := `[
		{
			"model": "NPS记录",
			"table": {
				"columns": ["提交时间", "NPS评分", "用户类型"],
				"readonly": true,
				"request_fields": [],
				"sample_rows": [{"提交时间": "2025-01-20", "NPS评分": "9", "用户类型": "推荐者"}]
			}
		},
		"title": "NPS 记录查询",
		"type": "table",
		"route": "nps_record_list.table"
	}]`
	result := reg.CallTool(context.Background(), "write_prd", map[string]interface{}{
		"project": map[string]interface{}{
			"name":                 "NPS 净推荐值系统",
			"code":                 "nps",
			"create_new_directory": true,
			"reason":               "NPS 是独立业务域",
		},
		"models": []map[string]interface{}{
			{
				"name": "NPS记录",
				"fields": []map[string]interface{}{
					{"name": "提交时间", "widget": "name:提交时间;type:datetime", "hide": "create,update"},
					{"name": "NPS评分", "widget": "name:NPS评分;type:number", "validate": "required"},
					{"name": "用户类型", "widget": "name:用户类型;type:select;options:推荐者,被动者,贬损者"},
				},
			},
		},
		"functions": functions,
	}, "/liubeiluo/ccc", "")
	if result.IsError {
		t.Fatalf("write_prd should repair prematurely closed JSON function object, got: %s", result.Content)
	}
	data, ok := result.Data.(writePRDResultData)
	if !ok {
		t.Fatalf("write_prd data type = %T, want writePRDResultData", result.Data)
	}
	if len(data.Functions) != 1 {
		t.Fatalf("functions len = %d, want 1", len(data.Functions))
	}
	if data.Functions[0].Title != "NPS 记录查询" || data.Functions[0].Route != "nps_record_list.table" {
		t.Fatalf("function not repaired correctly: %#v", data.Functions[0])
	}
}

func TestWritePRDToolRejectsInvalidPRDShape(t *testing.T) {
	reg := NewToolRegistry(nil)
	result := reg.CallTool(context.Background(), "write_prd", map[string]interface{}{
		"project": map[string]interface{}{
			"name":                 "会员收银系统",
			"code":                 "cashier",
			"create_new_directory": true,
		},
		"functions": []map[string]interface{}{
			{
				"title": "支付记录表",
				"type":  "table",
				"route": "cashier_payment_record_list",
				"table": map[string]interface{}{
					"columns": []string{"订单号"},
				},
			},
		},
	}, "/liubeiluo/ccc", "")
	if !result.IsError {
		t.Fatal("write_prd should reject invalid PRD")
	}
	data, ok := result.Data.(writePRDResultData)
	if !ok {
		t.Fatalf("write_prd data type = %T, want writePRDResultData", result.Data)
	}
	joined := strings.Join(data.Issues, "\n")
	for _, want := range []string{
		"project.reason",
		"route 必须以 .table 结尾",
		"sample_rows 至少需要 1 行示例数据",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("issues should contain %q, got %q", want, joined)
		}
	}
}

func TestWritePRDToolSchemaExposesStructuredData(t *testing.T) {
	reg := NewToolRegistry(nil)
	def := reg.tools["write_prd"].Definition()
	if def.Name != "write_prd" {
		t.Fatalf("tool name = %q", def.Name)
	}
	inputProps := def.InputSchema["properties"].(map[string]interface{})
	if _, ok := inputProps["project"]; !ok {
		t.Fatal("write_prd input schema should expose project")
	}
	if _, ok := inputProps["models"]; !ok {
		t.Fatal("write_prd input schema should expose models")
	}
	if _, ok := inputProps["functions"]; !ok {
		t.Fatal("write_prd input schema should expose functions")
	}
	rawInput, err := json.Marshal(def.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	if strings.Contains(string(rawInput), "go_source") {
		t.Fatalf("write_prd input schema should not expose go_source: %s", string(rawInput))
	}
	modelProps := inputProps["models"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
	fieldProps := modelProps["fields"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
	for _, name := range []string{"name", "widget", "validate", "hide", "description"} {
		if _, ok := fieldProps[name]; !ok {
			t.Fatalf("write_prd model field schema should expose %q", name)
		}
	}
	for _, name := range []string{"go_name", "json_name", "go_type", "gorm", "example", "options", "option_colors", "render_default"} {
		if _, ok := fieldProps[name]; ok {
			t.Fatalf("write_prd model field schema should not expose %q", name)
		}
	}

	outputProps := def.OutputSchema["properties"].(map[string]interface{})
	data, ok := outputProps["data"].(map[string]interface{})
	if !ok {
		t.Fatal("write_prd output schema should expose data")
	}
	dataProps := data["properties"].(map[string]interface{})
	for _, name := range []string{"kind", "project", "models", "functions", "acceptance_cases", "confirmation"} {
		if _, ok := dataProps[name]; !ok {
			t.Fatalf("write_prd data schema should expose %q", name)
		}
	}
	rawOutput, err := json.Marshal(def.OutputSchema)
	if err != nil {
		t.Fatalf("marshal output schema: %v", err)
	}
	if strings.Contains(string(rawOutput), "go_source") {
		t.Fatalf("write_prd output schema should not expose go_source: %s", string(rawOutput))
	}
}
