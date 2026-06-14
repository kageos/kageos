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
		"ctx.APICall",
		"ctx.GetRequestUser",
		"OnTableUpdateRowReq",
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

func TestPromptDocPathGuardsDisableRetiredSOPs(t *testing.T) {
	if IsPromptDocPath("/system/prompt/workspace/create-project") {
		t.Fatal("legacy workspace SOP should not be treated as prompt doc path")
	}

	retiredPath := func(leaf string) string {
		return "/system/prompt/" + retiredIntentPromptPackageCode + "/" + strings.TrimPrefix(leaf, "/")
	}
	for _, path := range []string{
		retiredPath("app-plan"),
		retiredPath("app-create"),
		retiredPath("modify/index"),
		retiredPath("modify/bugfix"),
	} {
		if IsPromptDocPath(path) {
			t.Fatalf("retired intent SOP should not be treated as prompt doc path: %s", path)
		}
		name, content := GetPromptDocContent(nil, path)
		if name != "" || content != "" {
			t.Fatalf("retired intent SOP should be unavailable: path=%s name=%q content=%q", path, name, content)
		}
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
		"/system/prompt/" + retiredIntentPromptPackageCode + "/publish-hub",
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
		"thumbnail:true;list_preview:true",
		"hide:\"create,update\"",
	} {
		if !strings.Contains(sdkContent, needle) {
			t.Fatalf("sdk readme should retain merged SDK knowledge %q", needle)
		}
	}
}

func TestProductManagerRoleRequiresPRDTablesAndConfirmation(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/roles/product-manager")
	if strings.Contains(content, "{{WORKSPACE_PRD_CONTRACT}}") {
		t.Fatalf("product_manager role doc should expand PRD contract marker: %q", content)
	}
	for _, needle := range []string{
		"产品经理 product_manager",
		"write_prd",
		"必须调用 `write_prd`",
		"`project/tables/forms/charts/rules`",
		"`search_fields` 只描述搜索参数",
		"`创建开始时间`、`创建结束时间`",
		"按记录创建时间范围查询",
		"用户筛选字段",
		"`handlers` 只表达表格行操作能力",
		"## 代表性输出示例",
		"禁止输出旧结构",
		"`models/functions/workflow/route/method/order/columns/sample_rows/preview_data/acceptance_cases/confirmation`",
		"禁止调用 `create_directory`",
		"app_developer",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("product_manager role doc should contain %q, got: %q", needle, content)
		}
	}
}

func TestAppDeveloperRoleExecutesConfirmedPRD(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/roles/app-developer")
	for _, needle := range []string{
		"应用开发工程师 app_developer",
		"只按已确认 PRD",
		"不重新设计 PRD",
		"不再次询问确认",
		"PRD JSON 作为唯一需求源",
		"tables.fields",
		"tables.search_fields",
		"不要因为搜索字段自动给 Go struct 增加同名业务列",
		"`创建开始时间`、`创建结束时间`",
		"`创建人` 是系统记录创建用户查询",
		"按可维护 Table、Form、只读记录 Table、Chart 的派生顺序生成",
		"禁止调用 `write_prd`",
		"写代码前必须先读取 1 到多个与当前需求匹配的案例",
		"/system/prompt/case_catalog/table/ticket",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"`write_doc`",
		"qa_engineer",
		"build_engineer",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("app_developer role doc should contain %q, got: %q", needle, content)
		}
	}
}

func TestAutomationOperatorRoleDocumentsScheduledAgentSOP(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/roles/automation-operator")
	for _, needle := range []string{
		"自动化操作员 automation_operator",
		"定时会话的典型场景",
		"长期数据维护",
		"情报/新闻日报",
		"周报/月报",
		"业务巡检",
		"定时会话 message 标准 SOP",
		"预期工具清单",
		"`run_table_search`",
		"`run_table_create` / `run_table_update`",
		"`web_search`",
		"`send_notification`",
		"定时会话/后台任务必须显式填写 `to_users`",
		"任务创建人 username",
		"首次基准记录",
		"模型库巡检示例 message",
		"场景化 owner 意识",
		"不是固定清单",
		"不要把示例规则机械套到所有任务",
		"跨资源工作流类",
		"其他目录函数或连接器",
		"可信度与写入规则",
		"证据不足",
		"禁止调用 `write_prd`、`create_directory`、`write_doc`、`write_go_file`、`search_replace_file`、`delete_file`、`build_workspace` 和任何业务 `run_*` 工具",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("automation_operator role doc should contain %q, got: %q", needle, content)
		}
	}
}

func TestExecutionRolesRetainPRDV2SearchRules(t *testing.T) {
	for docPath, needles := range map[string][]string{
		"/system/prompt/roles/qa-engineer": {
			"测试工程师 qa_engineer",
			"`search_fields` 里的核心筛选必须验证",
			"`创建开始时间/创建结束时间`",
			"`创建人/提交人/处理人/评分人/申请人`",
			"Form 提交后必须到 `target_table` 对应 Table 查询验证记录确实产生",
		},
		"/system/prompt/roles/app-operator": {
			"应用操作员 app_operator",
			"这是业务操作角色，不是测试角色",
			"`run_table_search`",
			"`run_table_create`",
			"`run_form_submit`",
			"不重新输出 PRD，不创建目录，不写文档，不写 Go 文件，不 build",
			"`run_python`",
			"定时会话/后台任务必须显式传 `to_users`",
		},
		"/system/prompt/roles/maintenance-engineer": {
			"应用维护工程师 maintenance_engineer",
			"`search_fields` 是查询请求字段",
			"`创建开始时间/创建结束时间`",
			"不要为了它们新增业务列",
			"裸写 `开始时间/结束时间` 只适合业务字段或 Chart 统计区间",
			"`write_doc`",
		},
		"/system/prompt/roles/build-engineer": {
			"构建修复工程师 build_engineer",
			"搜索字段不一定需要出现在 Go struct 中",
			"`创建开始时间/创建结束时间` 应修成系统创建时间查询逻辑",
			"`创建人` 应修成系统创建用户查询逻辑",
		},
		"/system/prompt/roles/reviewer": {
			"代码审查分析师 reviewer",
			"`project/tables/forms/charts/rules`",
			"`search_fields` 不应被误实现成业务模型字段",
			"`创建开始时间/创建结束时间/创建人` 应映射系统字段查询",
		},
	} {
		_, content := GetPromptDocContent(nil, docPath)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s should contain %q, got: %q", docPath, needle, content)
			}
		}
	}
}

func TestRoleToolDocsMentionRuntimeAllowedNotificationsAndExactRunTools(t *testing.T) {
	cases := map[string][]string{
		"/system/prompt/roles/qa-engineer": {
			"`run_table_search`",
			"`run_form_submit`",
			"`run_on_select_fuzzy`",
			"`send_notification`",
		},
		"/system/prompt/roles/data-operator": {
			"基础只读工具全角色可用",
			"`run_python`",
			"`send_notification`",
		},
		"/system/prompt/roles/platform-engineer": {
			"基础只读工具全角色可用",
			"`run_form_submit`",
			"`send_notification`",
		},
	}
	for docPath, needles := range cases {
		_, content := GetPromptDocContent(nil, docPath)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s should contain %q, got: %q", docPath, needle, content)
			}
		}
	}
}

func TestCaseCatalogDocsPreferPRDJSONV2(t *testing.T) {
	for _, docPath := range []string{
		"/system/prompt/case_catalog/form/pdf",
		"/system/prompt/case_catalog/formandtable/vote",
		"/system/prompt/case_catalog/tables/meeting",
	} {
		_, content := GetPromptDocContent(nil, docPath)
		for _, needle := range []string{
			"## 结构化 PRD JSON",
			`"schema_version": "prd.v2"`,
			`"rules"`,
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s should include PRD v2 JSON content %q, got: %q", docPath, needle, content)
			}
		}
		for _, forbidden := range []string{
			"旧版 PRD",
			"sample_rows",
			"preview_data",
			"acceptance_cases",
		} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s should not expose legacy PRD text %q, got: %q", docPath, forbidden, content)
			}
		}
	}
}
