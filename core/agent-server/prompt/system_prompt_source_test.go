package prompt

import (
	"strings"
	"testing"
)

func TestGetPromptDocContent_ForSDKDirectoryAndLeafDoc(t *testing.T) {
	sdkName, sdkContent := GetPromptDocContent(nil, "/system/prompt/sdk/agent-app-sdk-readme")
	if strings.TrimSpace(sdkName) == "" {
		t.Fatal("expected sdk readme doc name")
	}
	if !strings.Contains(sdkContent, "SDK 主入口") {
		t.Fatalf("expected sdk readme content, got: %q", sdkContent)
	}
	if !strings.Contains(sdkContent, "前端据此渲染列表字段") || strings.Contains(sdkContent, "第一张非空表") {
		t.Fatalf("expected sdk readme to explain AutoCrudTable/list rendering boundary, got: %q", sdkContent)
	}

	boundaryName, boundaryContent := GetPromptDocContent(nil, "/system/prompt/platform-capability-boundaries")
	if strings.TrimSpace(boundaryName) == "" {
		t.Fatal("expected platform capability boundaries doc name")
	}
	if !strings.Contains(boundaryContent, "平台能力边界") {
		t.Fatalf("expected platform capability boundaries content, got: %q", boundaryContent)
	}

	architectureName, architectureContent := GetPromptDocContent(nil, "/system/prompt/platform-function-architecture")
	if strings.TrimSpace(architectureName) == "" {
		t.Fatal("expected platform function architecture doc name")
	}
	if !strings.Contains(architectureContent, "Form/Table/Chart 组合架构") {
		t.Fatalf("expected platform function architecture content, got: %q", architectureContent)
	}

	widgetName, widgetContent := GetPromptDocContent(nil, "/system/prompt/sdk/widget-system")
	if strings.TrimSpace(widgetName) == "" {
		t.Fatal("expected widget system doc name")
	}
	if !strings.Contains(widgetContent, "SDK Widget 组件系统") {
		t.Fatalf("expected widget system content, got: %q", widgetContent)
	}

	formName, formContent := GetPromptDocContent(nil, "/system/prompt/sdk/form-submit-basic")
	if strings.TrimSpace(formName) == "" {
		t.Fatal("expected form submit basic doc name")
	}
	if !strings.Contains(formContent, "SDK Form 提交任务包") {
		t.Fatalf("expected form submit content, got: %q", formContent)
	}

	commonName, commonContent := GetPromptDocContent(nil, "/system/prompt/sdk/common-runtime-capabilities")
	if strings.TrimSpace(commonName) == "" {
		t.Fatal("expected common runtime capabilities doc name")
	}
	for _, want := range []string{
		"SDK 公共运行能力",
		"ctx.SendMessage",
		"ctx.APICall",
		"ctx.GetRequestUser",
		"OnTableUpdateRowReq",
		"定时任务",
		"事务和副作用顺序",
		"Python 和外部处理",
	} {
		if !strings.Contains(commonContent, want) {
			t.Fatalf("expected common runtime content to contain %q, got: %q", want, commonContent)
		}
	}

	tableName, tableContent := GetPromptDocContent(nil, "/system/prompt/sdk/table-crud-basic")
	if strings.TrimSpace(tableName) == "" {
		t.Fatal("expected table crud basic doc name")
	}
	if !strings.Contains(tableContent, "SDK Table CRUD 基础任务包") {
		t.Fatalf("expected table crud content, got: %q", tableContent)
	}

	comboTableFormName, comboTableFormContent := GetPromptDocContent(nil, "/system/prompt/sdk/combo-table-form")
	if strings.TrimSpace(comboTableFormName) == "" {
		t.Fatal("expected combo table form doc name")
	}
	if !strings.Contains(comboTableFormContent, "SDK Table/Form 组合任务包") {
		t.Fatalf("expected combo table form content, got: %q", comboTableFormContent)
	}

	comboName, comboContent := GetPromptDocContent(nil, "/system/prompt/sdk/combo-table-form-chart")
	if strings.TrimSpace(comboName) == "" {
		t.Fatal("expected combo table form chart doc name")
	}
	if !strings.Contains(comboContent, "SDK Table/Form/Chart 组合任务包") {
		t.Fatalf("expected combo content, got: %q", comboContent)
	}

	legacyName, legacyContent := GetPromptDocContent(nil, "/system/prompt/workspace/create-project")
	if legacyName != "" || legacyContent != "" {
		t.Fatalf("legacy workspace SOP docs should be unavailable, got name=%q content=%q", legacyName, legacyContent)
	}
}

func TestPromptDocCandidatePaths_PreferSeedActualPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/system/prompt/doc/workspace-env-template", want: "/system/prompt/doc/workspace-env-template.docs"},
		{path: "/system/prompt/mode/dev/config", want: "/system/prompt/mode/dev/config.docs"},
		{path: "/system/prompt/platform-capability-boundaries", want: "/system/prompt/platform-capability-boundaries.docs"},
		{path: "/system/prompt/platform-function-architecture", want: "/system/prompt/platform-function-architecture.docs"},
		{path: "/system/prompt/sdk/widget-system", want: "/system/prompt/sdk/widget-system.docs"},
		{path: "/system/prompt/sdk/common-runtime-capabilities", want: "/system/prompt/sdk/common-runtime-capabilities.docs"},
		{path: "/system/prompt/sdk/form-submit-basic", want: "/system/prompt/sdk/form-submit-basic.docs"},
		{path: "/system/prompt/sdk/table-crud-basic", want: "/system/prompt/sdk/table-crud-basic.docs"},
		{path: "/system/prompt/sdk/combo-table-form", want: "/system/prompt/sdk/combo-table-form.docs"},
		{path: "/system/prompt/sdk/combo-table-form-chart", want: "/system/prompt/sdk/combo-table-form-chart.docs"},
	}

	for _, tt := range tests {
		got := PromptDocCandidatePaths(tt.path)
		if len(got) == 0 {
			t.Fatalf("expected candidate paths for %s", tt.path)
		}
		if got[0] != tt.want {
			t.Fatalf("expected first candidate for %s to be %s, got %v", tt.path, tt.want, got)
		}
	}
}

func TestPromptDocCandidatePaths_DisablesLegacyWorkspaceSOP(t *testing.T) {
	if got := PromptDocCandidatePaths("/system/prompt/workspace/create-project"); len(got) != 0 {
		t.Fatalf("legacy workspace SOP should have no candidates, got %v", got)
	}
}

func TestScenarioTaskDocsContainClosedLoopSDKKnowledge(t *testing.T) {
	commonNeedles := []string{
		"必备 SDK 能力",
		"自定义搜索参数",
		"后置关联填充",
		"BuildFunctionUrlWithText",
		"display:\"scenes:list\"",
		"OnSelectFuzzy",
		"type:files",
		"DownloadFiles",
		"ResponseFiles",
		"落地目录和函数清单",
		"示例数据",
		"确认后我将创建目录",
	}
	for _, path := range []string{
		"/system/prompt/sdk/form-submit-basic",
		"/system/prompt/sdk/table-crud-basic",
		"/system/prompt/sdk/combo-table-form",
		"/system/prompt/sdk/combo-table-form-chart",
	} {
		_, content := GetPromptDocContent(nil, path)
		for _, needle := range commonNeedles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q", path, needle)
			}
		}
	}

	_, chartContent := GetPromptDocContent(nil, "/system/prompt/sdk/combo-table-form-chart")
	if !strings.Contains(chartContent, "Chart Request 和聚合") {
		t.Fatalf("combo table/form/chart doc missing Chart Request guidance")
	}
	for _, needle := range []string{
		"事实记录表",
		"OnTableAddRow",
		"OnTableUpdateRow",
		"OnTableDeleteRows",
		"前端就不会出现新增、编辑、删除入口",
	} {
		if !strings.Contains(chartContent, needle) {
			t.Fatalf("combo table/form/chart doc missing readonly callback guidance %q", needle)
		}
	}

	_, tableContent := GetPromptDocContent(nil, "/system/prompt/sdk/table-crud-basic")
	if !strings.Contains(tableContent, "三个回调都不配置") || !strings.Contains(tableContent, "收银记录") {
		t.Fatalf("table crud doc missing readonly callback rule")
	}
	if !strings.Contains(tableContent, "只读表也建议显式配置 `AutoCrudTable`") ||
		strings.Contains(tableContent, "第一张非空表") ||
		strings.Contains(tableContent, "降级") ||
		strings.Contains(tableContent, "没有 `AutoCrudTable`") {
		t.Fatalf("table crud doc should explain AutoCrudTable without exposing fallback details")
	}
	if !strings.Contains(tableContent, "不要写“评价对象ID”") ||
		!strings.Contains(tableContent, "用户通过下拉搜索对象名称") {
		t.Fatalf("table crud doc missing foreign-key fuzzy search guidance")
	}

	_, widgetContent := GetPromptDocContent(nil, "/system/prompt/sdk/widget-system")
	if !strings.Contains(widgetContent, "外键搜索也优先用 OnSelectFuzzy") ||
		!strings.Contains(widgetContent, "已关闭对象也可能有历史记录") {
		t.Fatalf("widget system doc missing foreign-key OnSelectFuzzy guidance")
	}
}
