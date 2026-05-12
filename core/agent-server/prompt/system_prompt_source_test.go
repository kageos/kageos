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

func TestPromptDocCandidatePaths_DisablesRetiredWorkflowSOP(t *testing.T) {
	retiredPath := func(leaf string) string {
		return "/system/prompt/" + retiredWorkflowPromptPackageCode + "/" + strings.TrimPrefix(leaf, "/")
	}
	for _, path := range []string{
		retiredPath("app-plan"),
		retiredPath("app-create"),
		retiredPath("modify/index"),
		retiredPath("modify/bugfix"),
	} {
		if IsPromptDocPath(path) {
			t.Fatalf("retired workflow SOP should not be treated as prompt doc path: %s", path)
		}
		if got := PromptDocCandidatePaths(path); len(got) != 0 {
			t.Fatalf("retired workflow SOP should have no candidates: path=%s got=%v", path, got)
		}
		name, content := GetPromptDocContent(nil, path)
		if name != "" || content != "" {
			t.Fatalf("retired workflow SOP should be unavailable: path=%s name=%q content=%q", path, name, content)
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
		"/system/prompt/" + retiredWorkflowPromptPackageCode + "/publish-hub",
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
		"`project/tables/forms/charts/workflow/rules`",
		"`search_fields` 只描述搜索参数",
		"`创建开始时间`、`创建结束时间`",
		"按记录创建时间范围查询",
		"用户筛选字段",
		"`handlers` 只表达表格行操作能力",
		"## 代表性输出示例",
		`"workflow"`,
		"禁止输出旧结构",
		"`models/functions/route/method/order/columns/sample_rows/preview_data/acceptance_cases/confirmation`",
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
		"按 `workflow` 数组顺序生成 Table/Form/Chart",
		"禁止调用 `write_prd`",
		"写代码前必须先读取 1 到多个与当前需求匹配的案例",
		"/system/prompt/case_catalog/table/ticket",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"qa_engineer",
		"build_engineer",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("app_developer role doc should contain %q, got: %q", needle, content)
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
			"不重新输出 PRD，不创建目录，不写 Go 文件，不 build",
		},
		"/system/prompt/roles/maintenance-engineer": {
			"应用维护工程师 maintenance_engineer",
			"`search_fields` 是查询请求字段",
			"`创建开始时间/创建结束时间`",
			"不要为了它们新增业务列",
			"裸写 `开始时间/结束时间` 只适合业务字段或 Chart 统计区间",
		},
		"/system/prompt/roles/build-engineer": {
			"构建修复工程师 build_engineer",
			"搜索字段不一定需要出现在 Go struct 中",
			"`创建开始时间/创建结束时间` 应修成系统创建时间查询逻辑",
			"`创建人` 应修成系统创建用户查询逻辑",
		},
		"/system/prompt/roles/reviewer": {
			"代码审查分析师 reviewer",
			"`project/tables/forms/charts/workflow/rules`",
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

func TestWorkflowEngineerRoleIncludesOrchestrationSOP(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/roles/workflow-engineer")
	for _, needle := range []string{
		"工作流编排工程师 workflow_engineer",
		"`workflow.v1`",
		"`search_resources`",
		"`search_tools`",
		"Graph Definition",
		"Expression Engine",
		"Node Executor Registry",
		"Run State Machine",
		"`form.submit`",
		"不要根据函数名、路由名、历史记忆或相似工具猜",
		"`missing_capabilities`",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("workflow_engineer role doc should contain %q, got: %q", needle, content)
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
			`"workflow"`,
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

func TestWorkflowCaseCatalogLoadsWorkflowDefinitionJSON(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/case_catalog/workflow/form_chain")
	for _, needle := range []string{
		"## Workflow Definition JSON",
		`"schema_version": "workflow.v1"`,
		`"type": "form.submit"`,
		`"$ref": "steps.extractText.output.提取的文本"`,
		"## 映射逻辑",
		"当前案例不能直接扩展为可运行 JSON",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("workflow form_chain case should contain %q, got: %q", needle, content)
		}
	}
}
