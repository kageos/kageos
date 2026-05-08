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

	commonName, commonContent := GetPromptDocContent(nil, "/system/prompt/sdk/reference/runtime-capabilities")
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

	buildName, buildContent := GetPromptDocContent(nil, "/system/prompt/sdk/reference/build-validation")
	if strings.TrimSpace(buildName) == "" {
		t.Fatal("expected build validation doc name")
	}
	if !strings.Contains(buildContent, "build_workspace") {
		t.Fatalf("expected build validation content, got: %q", buildContent)
	}

	platformAPIName, platformAPIContent := GetPromptDocContent(nil, "/system/prompt/sdk/reference/platform-api")
	if strings.TrimSpace(platformAPIName) == "" {
		t.Fatal("expected platform api doc name")
	}
	if !strings.Contains(platformAPIContent, "ctx.APICall") {
		t.Fatalf("expected platform api content, got: %q", platformAPIContent)
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
		{path: "/system/prompt/sdk/agent-app-sdk-readme", want: "/system/prompt/sdk/agent-app-sdk-readme.docs"},
		{path: "/system/prompt/sdk/reference", want: "/system/prompt/sdk/reference/index.docs"},
		{path: "/system/prompt/sdk/reference/runtime-capabilities", want: "/system/prompt/sdk/reference/runtime-capabilities.docs"},
		{path: "/system/prompt/sdk/reference/build-validation", want: "/system/prompt/sdk/reference/build-validation.docs"},
		{path: "/system/prompt/sdk/reference/platform-api", want: "/system/prompt/sdk/reference/platform-api.docs"},
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

func TestLeanPromptDocsMoveRedundantSDKTaskPacksOutOfSeed(t *testing.T) {
	for _, path := range []string{
		"/system/prompt/sdk/form-submit-basic",
		"/system/prompt/sdk/table-crud-basic",
		"/system/prompt/sdk/combo-table-form",
		"/system/prompt/sdk/combo-table-form-chart",
		"/system/prompt/sdk/form-table-chart-reference",
		"/system/prompt/sdk/widget-system",
		"/system/prompt/sdk/sdk",
		"/system/prompt/sdk/common-runtime-capabilities",
		"/system/prompt/sdk/build-validation-reference",
		"/system/prompt/sdk/platform-api-reference",
		"/system/prompt/platform-function-architecture",
		"/system/prompt/platform-cross-cutting-capabilities",
		"/system/prompt/intents/publish-hub",
		"/system/prompt/mode/dev/first_assistant",
		"/system/prompt/mode/dev/readme",
	} {
		name, content := GetPromptDocContent(nil, path)
		if name != "" || content != "" {
			t.Fatalf("redundant prompt doc should be unavailable: path=%s name=%q content=%q", path, name, content)
		}
	}

	_, sdkContent := GetPromptDocContent(nil, "/system/prompt/sdk/agent-app-sdk-readme")
	for _, needle := range []string{
		"WidgetLookupExample",
		"Table/Form/Chart 模式",
		"Chart 拆分规则（必读）",
		"不支持 `resp.Chart(chart1, chart2)`",
		"图表 `Metadata`",
		"BuildFunctionUrlWithText",
		"OnSelectFuzzy",
		"type:files",
		"hide:\"create,update\"",
	} {
		if !strings.Contains(sdkContent, needle) {
			t.Fatalf("sdk readme should retain merged SDK knowledge %q", needle)
		}
	}
}

func TestAppPlanSOPRequiresPRDTablesAndConfirmation(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/intents/app-plan")
	for _, needle := range []string{
		"应用设计 SOP",
		"`write_prd` 必须包含",
		"必须调用 `write_prd`",
		"不要只发纯文本 PRD",
		"「确认 PRD」按钮",
		"目录确认",
		"models 写法",
		"只描述“用户看到什么、怎么填、在哪些界面展示”",
		"每个字段只写 `name`、`widget`、`validate`、`hide`、`description`",
		"不需要列出 `ID`、`CreatedAt`、`UpdatedAt`、`DeletedAt`",
		"Table 写法",
		"Table 是 Table",
		"Table Request 是搜索/筛选请求",
		"列表模式",
		"用户可见预览只展示业务列表",
		"Form 写法",
		"必须明确 Request 和 Response",
		"请求（表单字段五列：字段 | 类型 | 必填 | 默认值 | 说明）",
		"支付记录表（cashier_payment_record_list.table）",
		"一个 Chart 行就是一个 `.chart` 路由",
		"Chart 也必须明确 Request 和 Response",
		"是否创建新目录",
		"确认后我再进入开发阶段",
		"禁止调用 `create_directory`、`write_go_file`、`build_workspace`",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("app.plan SOP should contain %q, got: %q", needle, content)
		}
	}
}

func TestAppCreateSOPExecutesConfirmedPRD(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/intents/app-create")
	for _, needle := range []string{
		"应用开发 SOP",
		"已确认的 PRD artifact",
		"不再重新设计 PRD",
		"不再二次询问确认",
		"如果用户只是提出新建系统但还没有确认 PRD，应切换到 `app.plan`",
		"把 PRD JSON 作为唯一需求源",
		"不要调用 `write_prd`",
		"根据 `models.fields` 自动生成 Go struct",
		"不要要求 PRD 提供字段 code、Go 类型或 `go_source`",
		"`models[].fields` 只要求 `name/widget/validate/hide/description`",
		"Table 有 Request",
		"Form 有 Request 和 Response",
		"Chart 有 Request 和 Response",
		"写代码前必须先读取 1 到多个与当前需求匹配的案例",
		"/system/prompt/case_catalog/table/ticket",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"/system/prompt/case_catalog/form/excelorcsv",
		"/system/prompt/intents/modify/chart-metric",
		"/system/prompt/sdk/reference/runtime-capabilities",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("app.create SOP should contain %q, got: %q", needle, content)
		}
	}
}
