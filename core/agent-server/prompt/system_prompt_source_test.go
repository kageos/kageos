package prompt

import (
	"os"
	"path/filepath"
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
	for _, want := range []string{
		"foreignKey + Preload",
		"DisableForeignKeyConstraintWhenMigrating",
		`Room *MeetingRoom gorm:"foreignKey:RoomID;references:ID"`,
		`Preload("Room")`,
		`json:"-" widget:"-"`,
	} {
		if !strings.Contains(sdkContent, want) {
			t.Fatalf("expected sdk readme to document Preload association pattern %q, got: %q", want, sdkContent)
		}
	}

	boundaryName, boundaryContent := GetPromptDocContent(nil, "/system/prompt/platform-capability-boundaries")
	if strings.TrimSpace(boundaryName) == "" {
		t.Fatal("expected platform capability boundaries doc name")
	}
	if !strings.Contains(boundaryContent, "平台能力边界") {
		t.Fatalf("expected platform capability boundaries content, got: %q", boundaryContent)
	}

	introName, introContent := GetPromptDocContent(nil, "/system/prompt/platform-introduction")
	if strings.TrimSpace(introName) == "" {
		t.Fatal("expected platform introduction doc name")
	}
	for _, want := range []string{
		"kageos 介绍与身份口径",
		"恰研智能（qiayan.ai）",
		"当前 kageos 核心采用 BSL 1.1",
		"Hub/企业版",
		"不要暗示第三方可以冒充官方 Hub",
	} {
		if !strings.Contains(introContent, want) {
			t.Fatalf("expected platform introduction content to contain %q, got: %q", want, introContent)
		}
	}

	usageName, usageContent := GetPromptDocContent(nil, "/system/prompt/platform-usage-and-philosophy")
	if strings.TrimSpace(usageName) == "" {
		t.Fatal("expected platform usage and philosophy doc name")
	}
	for _, want := range []string{
		"kageos 使用方式与产品理念",
		"目录是业务资产",
		"人机共用同一套能力",
		"平台管横切，应用管业务",
		"每天早上帮我生成日报",
	} {
		if !strings.Contains(usageContent, want) {
			t.Fatalf("expected platform usage and philosophy content to contain %q, got: %q", want, usageContent)
		}
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

	manifestName, manifestContent := GetPromptDocContent(nil, "/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask")
	if strings.TrimSpace(manifestName) == "" {
		t.Fatal("expected manifest/runbook/agent task doc name")
	}
	for _, want := range []string{
		"kageos_manifest.go / runbook.docs / AgentTask 写法",
		"`runbook.docs` 管“公司希望这类事情怎么处理”",
		"`AgentTask.Message` 管“Agent 到点后具体怎样安全执行”",
		"业务文档与技术执行的边界",
		"不要把业务人员变成 Agent 工程师",
		"不要反过来要求业务人员把这些技术细节补进 docs",
		"AgentTask 无人值守价值门禁",
		"提交时已经能确定",
		"结果回写位置",
		"文档优先的场景知识闭环",
		"运行态 Agent 可以通过 `read_file`、`edit_file`、`write_file`",
		"问题解决后直接删除对应内容",
		"不要先建一张与文档重复的知识表",
		"./docs/readme.docs",
		"不要为文档目录额外声明 `PackageContext`",
		"<./runbook.docs>",
		"<tool:send_notification>",
		"不要给 AgentTask 填 `Policy`",
	} {
		if !strings.Contains(manifestContent, want) {
			t.Fatalf("manifest/runbook/agent task doc should contain %q, got: %q", want, manifestContent)
		}
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

func TestPromptCaseCatalogUsesPreloadAssociations(t *testing.T) {
	root := filepath.Join("system", "prompt", "case_catalog")
	cases := map[string][]string{
		filepath.Join(root, "tables", "meeting", "prd.md"): {
			`Room     *MeetingRoom`,
			`gorm:"foreignKey:RoomID;references:ID"`,
			`Preload("Room")`,
		},
		filepath.Join(root, "tables", "hr", "prd.md"): {
			`Job           *HrJob`,
			`gorm:"foreignKey:JobID;references:ID"`,
			`Preload("Job")`,
		},
		filepath.Join(root, "formandtable", "vote", "prd.md"): {
			`Topic      *VoteTopic`,
			`Option        *VoteOption`,
			`Preload("Topic").Preload("Option")`,
		},
	}
	for path, needles := range cases {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("case catalog should use Preload associations, missing %q in %s", needle, path)
			}
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
		"ResolveChartBucket",
		"默认不会禁止细粒度",
		"`SeriesCount`：预计返回的系列数，不是数据库行数",
		"`dateExpr` / `groupExpr`",
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
		"时间趋势图的 `filters` 写清默认时间范围和粒度",
		"无人值守价值门禁",
		"提交—检查—退回—重新提交",
		"中小企业与可安装价值门禁",
		"文档优先的能力成长闭环",
		"不要让 Table 和 docs 保存同一份权威内容",
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
		"ResolveChartBucket",
		"`MaxValues`",
		"packageContext.AddDocs",
		"/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask",
		"提交当下可确定",
		"草稿/校验中",
		"后台新增价值",
		"./docs/readme.docs",
		"docs-first 闭环",
		"重复的 knowledge Table",
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
		"自动执行配置 automation_operator",
		"产品术语：数字员工",
		"“数字员工”“值守员工”是 `agent.session` Agent 任务的产品名称",
		"最终调用 `create_scheduled_agent_task`",
		"不要把“数字员工”理解成创建真实用户",
		"Agent 任务的典型场景",
		"长期数据维护",
		"情报/新闻日报",
		"周报/月报",
		"业务巡检",
		"先过无人值守价值门禁",
		"持续盯、等、查或协调",
		"不要替代提交时同步校验",
		"文档驱动的无人值守闭环",
		"只有正文明确“已启用”",
		"运行态 `app_operator` 可以用文件工具维护必要的 `.docs` 运行记忆",
		"Agent 任务执行说明（message）标准 SOP",
		"预期工具清单",
		"`run_table_search`",
		"`run_table_create` / `run_table_update`",
		"`web_search`",
		"`send_notification`",
		"可依赖 Agent 任务注入的默认通知对象并省略 `to_users`",
		"通知别人/多人或没有默认通知对象时显式填写 `to_users`",
		"首次基准记录",
		"模型库巡检示例 message",
		"场景化 owner 意识",
		"不是固定清单",
		"不要把示例规则机械套到所有任务",
		"跨资源工作流类",
		"其他目录函数或连接器",
		"工作台调用片段约定",
		"`<./daily_brief.form>`",
		"`<../shared/search_articles.form>`",
		"只有用户明确要求绑定其他工作空间时才使用 `</完整路径>`",
		"不可移植、复制后需重新绑定",
		"复制给工作台",
		"函数调用：",
		"`run_table_create`",
		"`run_table_update`",
		"`body = [{\"id\": 行ID, \"updates\": {...}}]`",
		"可信度与写入规则",
		"/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask",
		"<./runbook.docs>",
		"证据不足",
		"禁止调用 `write_prd`、`create_directory`、`write_file`、`edit_file`、`delete_file`、`build_workspace` 和任何业务 `run_*` 工具",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("automation_operator role doc should contain %q, got: %q", needle, content)
		}
	}
}

func TestRouterTreatsSmartEmployeeAsScheduledAgentTask(t *testing.T) {
	_, content := GetPromptDocContent(nil, "/system/prompt/roles/router")
	for _, needle := range []string{
		"创建 / 添加 / 配置 / 管理数字员工（值守员工）",
		"“数字员工”按 Agent 任务处理",
		"`create_scheduled_agent_task`",
		"“数字员工”是 Agent 任务的产品名称",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("router role doc should contain %q, got: %q", needle, content)
		}
	}
}

func TestDevModePromptRejectsDelayedValidationAutomation(t *testing.T) {
	provider := GetModeProvider("dev")
	if provider == nil {
		t.Fatal("dev mode provider missing")
	}
	content := provider.SystemPrompt(nil)
	for _, needle := range []string{
		"信息何时才可知",
		"必须同步校验",
		"提交—退回—重提",
		"产品界面和用户语言里的“数字员工”“值守员工”",
		"最终使用 `create_scheduled_agent_task`",
		"无人值守必须证明净价值",
		"结果要回写业务主表",
		"中小企业",
		"优先采用文档闭环",
		"组织知识默认文档优先",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("dev mode prompt should contain %q, got: %q", needle, content)
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
			"应用执行 app_operator",
			"这是业务操作角色，不是测试角色",
			"`run_table_search`",
			"`run_table_create`",
			"`run_form_submit`",
			"不重新输出 PRD，不创建目录，不写 Go 文件或普通文本文件，不 build",
			"`run_python`",
			"通知创建人、当前用户或“我”时可依赖默认通知对象并省略 `to_users`",
			"只使用明确“已启用”",
			"`file_name`/`code` 使用有意义的英文标识",
			"中文项目的工作台 `name` 使用清楚的中文名称",
			"问题解决后直接删除对应内容",
			"不增加“已解决”标记或历史清单",
		},
		"/system/prompt/roles/maintenance-engineer": {
			"应用维护工程师 maintenance_engineer",
			"/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask",
			"`search_fields` 是查询请求字段",
			"`创建开始时间/创建结束时间`",
			"不要为了它们新增业务列",
			"裸写 `开始时间/结束时间` 只适合业务字段或 Chart 统计区间",
			"ResolveChartBucket",
			"不要一刀切禁止细粒度",
			"packageContext.AddDocs",
			"不要创建本地 docs Go 子包",
			"docs-first",
			"待确认/待沉淀",
		},
		"/system/prompt/roles/build-engineer": {
			"构建修复工程师 build_engineer",
			"搜索字段不一定需要出现在 Go struct 中",
			"`创建开始时间/创建结束时间` 应修成系统创建时间查询逻辑",
			"`创建人` 应修成系统创建用户查询逻辑",
		},
		"/system/prompt/roles/reviewer": {
			"代码审查分析师 reviewer",
			"/system/prompt/platform-introduction",
			"/system/prompt/platform-usage-and-philosophy",
			"身份、公司、协议、Hub",
			"产品理念",
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

func TestWorkspaceRoleDocsContainHandoffGuidance(t *testing.T) {
	for _, docPath := range []string{
		"/system/prompt/roles/router",
		"/system/prompt/roles/product-manager",
		"/system/prompt/roles/app-developer",
		"/system/prompt/roles/maintenance-engineer",
		"/system/prompt/roles/qa-engineer",
		"/system/prompt/roles/app-operator",
		"/system/prompt/roles/automation-operator",
		"/system/prompt/roles/build-engineer",
		"/system/prompt/roles/data-operator",
		"/system/prompt/roles/platform-engineer",
		"/system/prompt/roles/reviewer",
	} {
		_, content := GetPromptDocContent(nil, docPath)
		for _, needle := range []string{"## 转岗指引", "留在", "交接给", "转交时必须携带"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s should contain handoff guidance marker %q, got: %q", docPath, needle, content)
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

func TestRepresentativeCaseCatalogDocsKeepSelectionBoundaries(t *testing.T) {
	casePaths := []string{
		"/system/prompt/case_catalog/agent/docs_service_desk",
		"/system/prompt/case_catalog/automation/site_monitor",
		"/system/prompt/case_catalog/transaction/consumable_inventory",
		"/system/prompt/case_catalog/public/service_booking",
		"/system/prompt/case_catalog/agent/hybrid_crm_followup",
	}
	for _, docPath := range casePaths {
		_, content := GetPromptDocContent(nil, docPath)
		for _, needle := range []string{
			`"schema_version": "prd.v2"`,
			"## 三、适用场景与选择边界",
			"### 适用场景",
			"### 不适用场景",
			"### 与相邻案例怎么选",
			"### 五分钟价值路径",
			"### 不要照搬",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s should keep case selection marker %q, got: %q", docPath, needle, content)
			}
		}
	}

	_, index := GetPromptDocContent(nil, "/system/prompt/case_catalog")
	for _, needle := range []string{
		"## 容易混淆的案例怎么选",
		"Form schedule",
		"AgentTask",
		"事实与建议分开",
		"事务、行锁和不可变流水",
		"公开面与内部面分开",
	} {
		if !strings.Contains(index, needle) {
			t.Fatalf("case catalog index should keep selection guidance %q, got: %q", needle, index)
		}
	}
}

func TestPromptDistinguishesFunctionAndAgentTaskDefaultState(t *testing.T) {
	_, sdk := GetPromptDocContent(nil, "/system/prompt/sdk/agent-app-sdk-readme")
	for _, needle := range []string{
		"普通 `FormSchedule` 是安装后即可运行的确定性业务自动化",
		"必须显式写 `Enabled: true`",
		"数字员工 `AgentTask` 可以显式写 `Enabled: false`",
		"运行态状态不应被代码更新覆盖",
	} {
		if !strings.Contains(sdk, needle) {
			t.Fatalf("SDK prompt should keep schedule default-state guidance %q", needle)
		}
	}

	_, manifest := GetPromptDocContent(nil, "/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask")
	if !strings.Contains(manifest, "数字员工默认显式写 `Enabled: false`") {
		t.Fatal("AgentTask manifest guide should require user-enabled digital employees")
	}
}
