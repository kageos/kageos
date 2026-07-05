package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
)

func TestBuildChangeRoleLoadsPlanDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRoleProductManager,
		UserInput:  "帮我搞个 NPS 管理系统",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != WorkspaceRoleProductManager || got.RoleID != WorkspaceRoleProductManager {
		t.Fatalf("current role=%s role_id=%s want %s", got.CurrentRole, got.RoleID, WorkspaceRoleProductManager)
	}
	if got.DisplayName != "产品经理" {
		t.Fatalf("display name=%s want 产品经理", got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/product-manager") {
		t.Fatalf("required docs should include product manager role doc: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/case_catalog") {
		t.Fatalf("product_manager should include case catalog index: %v", got.RequiredDocs)
	}
	for _, heavyCase := range []string{
		"/system/prompt/case_catalog/formandtable/vote",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"/system/prompt/case_catalog/table/ticket",
	} {
		if containsWorkspaceRoleString(got.RequiredDocs, heavyCase) {
			t.Fatalf("product_manager should not auto-inject full case %s before PRD confirmation: %v", heavyCase, got.RequiredDocs)
		}
	}
	for _, removed := range []string{
		"/system/prompt/sdk/widget-reference",
		"/system/prompt/sdk/build-validation-reference",
		"/system/prompt/sdk/workbench-tools-sdk-relationship",
		"/system/prompt/platform-overview",
		"/system/prompt/platform-capability-boundaries",
	} {
		if containsWorkspaceRoleString(got.RequiredDocs, removed) {
			t.Fatalf("product_manager should not inject heavy doc %s: %v", removed, got.RequiredDocs)
		}
	}
	if len(got.LoadedDocs) == 0 {
		t.Fatal("expected loaded docs")
	}
}

func TestChangeRoleToolSchemaMentionsAutomationOperator(t *testing.T) {
	def := (&ChangeRoleTool{}).Definition()
	properties := def.InputSchema["properties"].(map[string]interface{})
	targetRole := properties["target_role"].(map[string]interface{})
	description, _ := targetRole["description"].(string)
	if !strings.Contains(description, WorkspaceRoleAutomationOperator) {
		t.Fatalf("target_role schema should mention %s, description=%q", WorkspaceRoleAutomationOperator, description)
	}
	if !strings.Contains(description, WorkspaceRoleRouter) {
		t.Fatalf("target_role schema should mention %s, description=%q", WorkspaceRoleRouter, description)
	}
	executeDirectory := properties["execute_directory"].(map[string]interface{})
	executeDescription, _ := executeDirectory["description"].(string)
	for _, want := range []string{
		"主执行目录/绑定目录",
		"其他空间函数或连接器函数完整路径",
		"权限由平台统一判断",
	} {
		if !strings.Contains(executeDescription, want) {
			t.Fatalf("execute_directory schema should contain %q, description=%q", want, executeDescription)
		}
	}
}

func TestChangeRoleInvalidRoleHintMentionsAutomationOperator(t *testing.T) {
	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		Args: map[string]interface{}{
			"target_role":       "unknown_role",
			"execute_directory": "/system/x_world/vote",
		},
	})
	if !res.IsError {
		t.Fatalf("unknown target_role should return error: %#v", res)
	}
	for _, roleID := range workspaceStandardRoleIDs() {
		if !strings.Contains(res.Content, roleID) {
			t.Fatalf("invalid role hint should mention %s, content=%q", roleID, res.Content)
		}
	}
}

func TestBuildChangeRoleLoadsRouterHandbook(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleQAEngineer,
		TargetRole:       WorkspaceRoleRouter,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"QA 测试失败后不知道该切维护还是构建修复"},
		ResetContext:     true,
	})
	if got.CurrentRole != WorkspaceRoleRouter || got.DisplayName != "执行路由手册" {
		t.Fatalf("role=%s display=%s want router/执行路由手册", got.CurrentRole, got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/router") {
		t.Fatalf("router should require handbook, docs=%v", got.RequiredDocs)
	}
	if len(got.LoadedDocs) == 0 {
		t.Fatalf("router should load handbook content, docs=%#v", got.LoadedDocs)
	}
	for _, want := range []string{"3 步急救流程", "立即决策流程", "门禁错误的处理公式", "换挡前自检"} {
		if !strings.Contains(got.LoadedDocs[0].Content, want) {
			t.Fatalf("router handbook should contain %q, docs=%#v", want, got.LoadedDocs)
		}
	}
	if !containsWorkspaceRoleString(got.AllowedNextTools, "change_role") ||
		containsWorkspaceRoleString(got.AllowedNextTools, "edit_file") {
		t.Fatalf("router allowed tools should be read-only plus change_role, tools=%v", got.AllowedNextTools)
	}
}

func TestBuildChangeRoleLoadsCreateDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRoleAppDeveloper,
		UserInput:  "已确认 PRD，开始创建目录和生成代码",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != WorkspaceRoleAppDeveloper || got.RoleID != WorkspaceRoleAppDeveloper {
		t.Fatalf("current role=%s role_id=%s want %s", got.CurrentRole, got.RoleID, WorkspaceRoleAppDeveloper)
	}
	if got.DisplayName != "应用开发工程师" {
		t.Fatalf("display name=%s want 应用开发工程师", got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/app-developer") {
		t.Fatalf("required docs should include app developer role doc: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("required docs should include SDK readme: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/sdk/reference/kageos-manifest-runbook-agenttask") {
		t.Fatalf("required docs should include manifest/runbook/agent task guide: %v", got.RequiredDocs)
	}
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_prd") {
		t.Fatalf("app_developer should not plan PRD again, tools=%v", got.AllowedNextTools)
	}
	if !containsWorkspaceRoleString(got.RuntimeContract.DoneWhen, "build_workspace 成功、已交接 qa_engineer 并完成核心函数测试") {
		t.Fatalf("app_developer should expose done_when contract, runtime=%#v", got.RuntimeContract)
	}
	if len(got.RuntimeContract.Hooks) == 0 {
		t.Fatalf("app_developer should expose lifecycle hooks, runtime=%#v", got.RuntimeContract)
	}
	if got.RuntimeContract.Hooks[0].ImplementationStatus == "" {
		t.Fatalf("runtime hooks should expose implementation status, hooks=%#v", got.RuntimeContract.Hooks)
	}
}

func TestBuildChangeRoleLoadsBuildRepairDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole:       WorkspaceRoleBuildEngineer,
		ExecuteDirectory: "/u/app/nps",
		TaskContext:      []string{"build_workspace 失败，需要修复 schema"},
	})
	for _, doc := range []string{
		"/system/prompt/roles/build-engineer",
		"/system/prompt/sdk/agent-app-sdk-readme",
		"/system/prompt/sdk/reference/build-validation",
	} {
		if !containsWorkspaceRoleString(got.RequiredDocs, doc) {
			t.Fatalf("build engineer should include %s, docs=%v", doc, got.RequiredDocs)
		}
	}
}

func TestBuildChangeRoleDoesNotInferFromUserInput(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		UserInput: "帮我搞个 NPS 管理系统",
	})
	if got.CurrentRole != WorkspaceRoleReviewer {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleReviewer)
	}
	if containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/product-manager") ||
		containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/app-developer") {
		t.Fatalf("change_role should not infer product manager/developer from user_input: %v", got.RequiredDocs)
	}
	for _, doc := range []string{
		"/system/prompt/platform-introduction",
		"/system/prompt/platform-usage-and-philosophy",
		"/system/prompt/platform-capability-boundaries",
	} {
		if !containsWorkspaceRoleString(got.RequiredDocs, doc) {
			t.Fatalf("reviewer should load introduction/usage/boundary doc %s, docs=%v", doc, got.RequiredDocs)
		}
	}
	loadedIntroDoc := false
	loadedUsageDoc := false
	for _, doc := range got.LoadedDocs {
		if doc.Path == "/system/prompt/platform-introduction" && strings.Contains(doc.Content, "Kageos 介绍与身份口径") {
			loadedIntroDoc = true
		}
		if doc.Path == "/system/prompt/platform-usage-and-philosophy" && strings.Contains(doc.Content, "Kageos 使用方式与产品理念") {
			loadedUsageDoc = true
		}
	}
	if !loadedIntroDoc || !loadedUsageDoc {
		t.Fatalf("reviewer should return platform introduction and usage doc content, intro=%v usage=%v loaded=%#v", loadedIntroDoc, loadedUsageDoc, got.LoadedDocs)
	}
}

func TestBuildChangeRoleKeepsCurrentWhenTargetMissing(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole: WorkspaceRoleAppDeveloper,
		UserInput:   "build_workspace 报错",
	})
	if got.CurrentRole != WorkspaceRoleAppDeveloper {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleAppDeveloper)
	}
}

func TestBuildChangeRoleSwitchesAndCarriesSummary(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:  WorkspaceRoleAppDeveloper,
		TargetRole:   WorkspaceRoleQAEngineer,
		TaskSummary:  "NPS 系统 build 已通过",
		ResetContext: true,
	})
	if !got.Switched {
		t.Fatal("expected role switch")
	}
	if got.CurrentRole != WorkspaceRoleQAEngineer {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleQAEngineer)
	}
	if got.ContextPolicy == "" {
		t.Fatal("expected context policy")
	}
	if !strings.Contains(got.Reason, "符合推荐流转") || !strings.Contains(got.Reason, "build 成功") {
		t.Fatalf("expected recommended transition reason, got %q", got.Reason)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/qa-engineer") {
		t.Fatalf("required docs should include qa engineer doc: %v", got.RequiredDocs)
	}
}

func TestBuildChangeRoleUsesStandardHandoffBlocksAndDirectoryFallback(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole: WorkspaceRoleBuildEngineer,
		TargetRole:  WorkspaceRoleQAEngineer,
		TaskContext: []string{
			"上一阶段 build_workspace 已通过",
			"用户要验证 NPS 表单提交和趋势图",
			"特殊 case：创建开始时间/创建结束时间必须按系统创建时间筛选",
		},
		KeyInformation: []string{"构建版本 v4", "Form: /liubeiluo/nps/submit_score", "Chart: /liubeiluo/nps/nps_trend"},
		References:     []string{"/system/prompt/roles/qa-engineer", "nps_submit.go"},
		ResetContext:   true,
	}, "/liubeiluo/nps")
	if got.ExecuteDirectory != "/liubeiluo/nps" || got.Directory != "/liubeiluo/nps" || got.Handoff.ExecuteDirectory != "/liubeiluo/nps" {
		t.Fatalf("handoff should use fallback execute directory, got %#v", got.Handoff)
	}
	if !strings.Contains(got.ContextPolicy, "主执行目录/绑定目录为 /liubeiluo/nps") {
		t.Fatalf("context policy should pin execute directory, got %q", got.ContextPolicy)
	}
	if len(got.Handoff.TaskContext) != 3 || !strings.Contains(strings.Join(got.Handoff.TaskContext, "；"), "趋势图") {
		t.Fatalf("handoff task context not preserved: %#v", got.Handoff)
	}
	if !containsWorkspaceRoleString(got.Handoff.KeyInformation, "构建版本 v4") || !containsWorkspaceRoleString(got.Handoff.References, "nps_submit.go") {
		t.Fatalf("handoff key info/references not preserved: %#v", got.Handoff)
	}
	if got.HandoffPacket.Version != workspaceRoleHandoffPacketVersion ||
		got.HandoffPacket.SourceRole != WorkspaceRoleBuildEngineer ||
		got.HandoffPacket.TargetRole != WorkspaceRoleQAEngineer ||
		got.HandoffPacket.ExecuteDirectory != "/liubeiluo/nps" {
		t.Fatalf("typed handoff packet metadata wrong: %#v", got.HandoffPacket)
	}
	if !containsWorkspaceRoleString(got.HandoffPacket.TaskContext, "上一阶段 build_workspace 已通过") ||
		!containsWorkspaceRoleString(got.HandoffPacket.KeyInformation, "构建版本 v4") ||
		!containsWorkspaceRoleString(got.HandoffPacket.References, "nps_submit.go") {
		t.Fatalf("typed handoff packet should mirror standard four blocks, got %#v", got.HandoffPacket)
	}
	if got.HandoffPacket.BuildDiagnostics != nil {
		t.Fatalf("QA handoff packet should not carry build diagnostics, got %#v", got.HandoffPacket.BuildDiagnostics)
	}
}

func TestBuildChangeRoleNormalizesNewAppDeveloperExecuteDirectoryToWorkspaceRoot(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleProductManager,
		TargetRole:       WorkspaceRoleAppDeveloper,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"PRD已确认，进入开发阶段", "目标：创建投票系统目录和 Go 代码"},
		KeyInformation:   []string{"项目：投票系统 (vote)", "3个Table和2个Form"},
		References:       []string{"/system/prompt/roles/app-developer", "/system/x_world/ticket_management", "/system/x_world/vote"},
		ResetContext:     true,
	}, "/system/x_world/ticket_management")
	if got.ExecuteDirectory != "/system/x_world" || got.Handoff.ExecuteDirectory != "/system/x_world" {
		t.Fatalf("new app developer should execute from workspace root, got %#v", got.Handoff)
	}
	if !containsWorkspaceRoleString(got.Handoff.KeyInformation, "新建应用目标目录：/system/x_world/vote") {
		t.Fatalf("handoff should preserve target new directory as key info: %#v", got.Handoff.KeyInformation)
	}
	if containsWorkspaceRoleString(got.Handoff.References, "/system/x_world/ticket_management") {
		t.Fatalf("handoff should prune stale sibling app references: %#v", got.Handoff.References)
	}
	if !containsWorkspaceRoleString(got.Handoff.References, "/system/x_world/vote") ||
		!containsWorkspaceRoleString(got.Handoff.References, "/system/prompt/roles/app-developer") {
		t.Fatalf("handoff should keep target directory and role docs: %#v", got.Handoff.References)
	}
	if !strings.Contains(got.ContextPolicy, "主执行目录/绑定目录为 /system/x_world") {
		t.Fatalf("context policy should pin workspace root execute directory, got %q", got.ContextPolicy)
	}
}

func TestBuildChangeRoleDoesNotNormalizeAppDeveloperForBusinessOperation(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleProductManager,
		TargetRole:       WorkspaceRoleAppDeveloper,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"用户要直接创建四大古都投票", "直接通过表单提交创建投票主题和选项"},
		References:       []string{"/system/prompt/roles/app-developer", "/system/x_world/vote"},
		ResetContext:     true,
	}, "/system/x_world/vote")
	if got.ExecuteDirectory != "/system/x_world/vote" || got.Handoff.ExecuteDirectory != "/system/x_world/vote" {
		t.Fatalf("business operation misrouted to app_developer should not be rewritten as new app development: %#v", got.Handoff)
	}
	for _, item := range got.Handoff.KeyInformation {
		if strings.Contains(item, "新建应用目标目录") {
			t.Fatalf("business operation should not receive new-app target directory hint: %#v", got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleAddsRoleSpecificNextStepAdvice(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleBuildEngineer,
		TargetRole:       WorkspaceRoleQAEngineer,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"build 已通过，进入测试"},
		KeyInformation:   []string{"重点验证投票主题创建、提交投票和查看结果"},
		ResetContext:     true,
	})
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"search(full_code_path=execute_directory",
		"主数据/配置表 -> Form 提交 -> 目标记录表 -> Chart/结果查询",
		"参数、数据、schema、业务 bug 或构建问题",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("QA handoff advice should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleRunsAppOperatorCapabilityHook(t *testing.T) {
	oldSearchFunctions := workspaceRoleHookSearchFunctions
	t.Cleanup(func() {
		workspaceRoleHookSearchFunctions = oldSearchFunctions
	})
	var gotReq *dto.SearchFunctionsReq
	workspaceRoleHookSearchFunctions = func(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
		gotReq = req
		return &dto.SearchFunctionsResp{
			Functions: []*dto.FunctionSearchResult{
				{
					Name:         "投票主题",
					Code:         "vote_topic_list",
					FullCodePath: "/system/x_world/vote/vote_topic_list.table",
					TemplateType: "table",
					Callbacks:    []string{"OnTableAddRow"},
					Schema: functionschema.NewTable(
						[]*widget.Field{testSearchField("topic_title", "主题标题", "input", nil, "")},
						[]*widget.Field{testSearchField("topic_title", "主题标题", "input", nil, "required")},
						[]string{"OnTableAddRow"},
					),
				},
				{
					Name:         "提交投票",
					Code:         "vote_submit",
					FullCodePath: "/system/x_world/vote/vote_submit.form",
					TemplateType: "form",
					Schema: functionschema.NewForm(
						[]*widget.Field{testSearchField("option_id", "选项", "select", nil, "required")},
						nil,
						nil,
					),
				},
				{
					Name:         "其他应用函数",
					FullCodePath: "/system/x_world/ticket/ticket_list.table",
					TemplateType: "table",
				},
			},
		}, nil
	}

	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole:       WorkspaceRoleAppOperator,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"用户要创建一个四大古都投票"},
	}, "/system/x_world/vote")

	if gotReq == nil || gotReq.FullCodePath != "/system/x_world/vote" || gotReq.User != "" || gotReq.App != "" || gotReq.Keyword != "" {
		t.Fatalf("unexpected function search request: %#v", gotReq)
	}
	if got.AppCapabilities == nil || got.AppCapabilities.Status != "ok" {
		t.Fatalf("expected app capability snapshot, got %#v", got.AppCapabilities)
	}
	if got.AppCapabilities.TotalFunctions != 2 || got.AppCapabilities.Counts.Tables != 1 || got.AppCapabilities.Counts.Forms != 1 {
		t.Fatalf("unexpected app capability counts: %#v", got.AppCapabilities)
	}
	if len(got.ExecutedHooks) != 1 ||
		got.ExecutedHooks[0].ID != workspaceRoleHookAppOperatorCapabilities ||
		got.ExecutedHooks[0].Stage != workspaceRoleHookStageBeforeEnter {
		t.Fatalf("expected app operator before_enter hook record, got %#v", got.ExecutedHooks)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"当前应用能力快照",
		"/system/x_world/vote/vote_topic_list.table",
		"run_table_create",
		"/system/x_world/vote/vote_submit.form",
		"search(full_code_path=change_role.execute_directory",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("app operator handoff key info should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleAppOperatorCapabilityHookDoesNotBlockOnSearchError(t *testing.T) {
	oldSearchFunctions := workspaceRoleHookSearchFunctions
	t.Cleanup(func() {
		workspaceRoleHookSearchFunctions = oldSearchFunctions
	})
	workspaceRoleHookSearchFunctions = func(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
		return nil, errors.New("service tree unavailable")
	}

	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole:       WorkspaceRoleAppOperator,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"用户要查询投票结果"},
	}, "/system/x_world/vote")

	if got.AppCapabilities == nil || got.AppCapabilities.Status != "error" {
		t.Fatalf("expected error snapshot, got %#v", got.AppCapabilities)
	}
	if got.CurrentRole != WorkspaceRoleAppOperator || len(got.ExecutedHooks) != 1 {
		t.Fatalf("change_role should still switch to app_operator and record hook, got %#v", got)
	}
	if !strings.Contains(strings.Join(got.Handoff.KeyInformation, "；"), "search(full_code_path=change_role.execute_directory") {
		t.Fatalf("error snapshot should carry search fallback, got %#v", got.Handoff.KeyInformation)
	}
}

func TestBuildChangeRoleAddsFailureAdviceBeforeRetrying(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleAppDeveloper,
		TargetRole:       WorkspaceRoleBuildEngineer,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"build 失败，多次重写后仍然报错"},
		References:       []string{"/system/prompt/sdk/agent-app-sdk-readme", "/system/prompt/case_catalog/formandtable/vote"},
		ResetContext:     true,
	})
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"不要猜不存在的 API",
		"不要继续同一方案重试",
		"先读取 references 中的 SDK、案例、源码或日志",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("failure handoff advice should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleRunsBuildEngineerDiagnosticsHook(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleAppDeveloper,
		TargetRole:       WorkspaceRoleBuildEngineer,
		ExecuteDirectory: "/system/x_world/inventory",
		TaskContext: []string{
			"build_workspace 失败，进入构建修复阶段",
			"app startup failed: SDK schema compile failed: router /inventory/purchase_inbound_list.table table schema decode failed",
		},
		KeyInformation: []string{
			`field SupplierName (supplier_name): widget "select" requires options or OnSelectFuzzyMap entry`,
			`field CreatedBy (created_by): audit field "created_by" hide tag must be "create,update", got ""`,
		},
		References:   []string{"/system/prompt/sdk/reference/build-validation"},
		ResetContext: true,
	}, "/system/x_world/inventory")

	if got.BuildDiagnostics == nil {
		t.Fatalf("expected build diagnostics, got %#v", got)
	}
	for _, want := range []string{"schema_validation", "select_options", "audit_field"} {
		if !containsWorkspaceRoleString(got.BuildDiagnostics.Categories, want) {
			t.Fatalf("expected diagnostics category %q, got %#v", want, got.BuildDiagnostics.Categories)
		}
	}
	if !containsWorkspaceRoleString(got.BuildDiagnostics.Routers, "/inventory/purchase_inbound_list.table") {
		t.Fatalf("expected router in diagnostics, got %#v", got.BuildDiagnostics.Routers)
	}
	if !workspaceBuildDiagnosticsHasFieldIssue(got.BuildDiagnostics, "CreatedBy", "created_by") {
		t.Fatalf("expected CreatedBy field issue, got %#v", got.BuildDiagnostics.FieldIssues)
	}
	if len(got.ExecutedHooks) != 1 ||
		got.ExecutedHooks[0].ID != workspaceRoleHookBuildEngineerDiagnostics ||
		got.ExecutedHooks[0].Stage != workspaceRoleHookStageBeforeEnter {
		t.Fatalf("expected build engineer diagnostics hook record, got %#v", got.ExecutedHooks)
	}
	if got.HandoffPacket.BuildDiagnostics == nil {
		t.Fatalf("typed build engineer packet should carry build diagnostics, got %#v", got.HandoffPacket)
	}
	if !containsWorkspaceRoleString(got.HandoffPacket.BuildDiagnostics.Categories, "schema_validation") ||
		!containsWorkspaceRoleString(got.HandoffPacket.References, "/system/prompt/sdk/reference/build-validation") {
		t.Fatalf("typed build engineer packet should carry diagnostics and required docs, got %#v", got.HandoffPacket)
	}
	if len(got.HandoffPacket.ExecutedHooks) != 1 ||
		got.HandoffPacket.ExecutedHooks[0].ID != workspaceRoleHookBuildEngineerDiagnostics {
		t.Fatalf("typed build engineer packet should expose executed hooks, got %#v", got.HandoffPacket.ExecutedHooks)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"构建诊断",
		"错误类型=schema_validation、audit_field、select_options",
		"构建修复必读资料",
		"同类错误第二次出现前",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("build engineer handoff key info should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleRunsMaintenanceScopeHook(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleQAEngineer,
		TargetRole:       WorkspaceRoleMaintenanceEngineer,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"投票提交测试失败，需要修复提交逻辑"},
		KeyInformation:   []string{"失败函数：/system/x_world/vote/vote_submit.form", "相关文件：/system/x_world/vote/vote.go"},
		ResetContext:     true,
	})
	if len(got.ExecutedHooks) != 1 ||
		got.ExecutedHooks[0].ID != workspaceRoleHookMaintenanceScope ||
		got.ExecutedHooks[0].Stage != workspaceRoleHookStageBeforeEnter {
		t.Fatalf("expected maintenance scope hook record, got %#v", got.ExecutedHooks)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"维护范围",
		"execute_directory=/system/x_world/vote",
		"/system/x_world/vote/vote_submit.form",
		"只读取、修改、构建该目录或其子目录",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("maintenance handoff key info should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleRunsQABeforeEnterSchemaHook(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:      WorkspaceRoleBuildEngineer,
		TargetRole:       WorkspaceRoleQAEngineer,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"build 已通过，进入测试"},
		KeyInformation:   []string{"Form: /system/x_world/vote/vote_submit.form", "Table: /system/x_world/vote/vote_topic_list.table"},
		ResetContext:     true,
	})
	if len(got.ExecutedHooks) != 1 ||
		got.ExecutedHooks[0].ID != workspaceRoleHookQABeforeEnterSchema ||
		got.ExecutedHooks[0].Stage != workspaceRoleHookStageBeforeEnter {
		t.Fatalf("expected QA before_enter hook record, got %#v", got.ExecutedHooks)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	for _, want := range []string{
		"测试范围",
		"execute_directory=/system/x_world/vote",
		"候选测试函数",
		"/system/x_world/vote/vote_submit.form",
		"search(full_code_path=execute_directory",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("QA handoff key info should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
	}
}

func TestBuildChangeRoleKeepsRichSummaryAcrossRoles(t *testing.T) {
	summary := strings.Repeat("已确认范围=提交评分 Form、满意度记录只读表、NPS 趋势图；", 10) +
		"关键约束=记录表由 Form 产生，不开放手工新增；验证重点=门店筛选和日期趋势。"
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:    WorkspaceRoleQAEngineer,
		TargetRole:     WorkspaceRoleMaintenanceEngineer,
		TaskSummary:    summary,
		ReferenceDocs:  []string{"/system/prompt/roles/maintenance-engineer", "/system/prompt/sdk/agent-app-sdk-readme"},
		ReferenceFiles: []string{"nps_submit.go", "nps_chart.go"},
		ResetContext:   true,
	})
	if !got.Switched || got.CurrentRole != WorkspaceRoleMaintenanceEngineer {
		t.Fatalf("expected switch to maintenance engineer, got %#v", got)
	}
	if !strings.Contains(got.ContextPolicy, "关键约束=记录表由 Form 产生") || !strings.Contains(got.ContextPolicy, "验证重点=门店筛选和日期趋势") {
		t.Fatalf("context policy should keep rich summary, got %q", got.ContextPolicy)
	}
	if !strings.Contains(got.ContextPolicy, "参考文档=/system/prompt/roles/maintenance-engineer") || !strings.Contains(got.ContextPolicy, "参考文件=nps_submit.go") {
		t.Fatalf("context policy should keep references, got %q", got.ContextPolicy)
	}
	if len(got.ReferenceDocs) != 2 || len(got.ReferenceFiles) != 2 {
		t.Fatalf("change_role should return explicit references, got %#v", got)
	}
}

func TestBuildChangeRoleReportsBaseReadOnlyToolsForEveryRole(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRolePlatformEngineer,
		UserInput:  "全链路测试 openapi",
	})
	for _, tool := range []string{"read_dir", "read_file", "read_app_log", "search"} {
		if !containsWorkspaceRoleString(got.AllowedNextTools, tool) {
			t.Fatalf("platform engineer should report base read-only tool %s, tools=%v", tool, got.AllowedNextTools)
		}
	}
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_file") {
		t.Fatalf("platform engineer should not report write_file, tools=%v", got.AllowedNextTools)
	}
}

func TestChangeRoleRejectsRetiredOrMalformedRoleID(t *testing.T) {
	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		Args: map[string]interface{}{
			"target_role": "app_operate_test",
		},
	})
	if !res.IsError {
		t.Fatalf("expected invalid target_role to fail, got %#v", res)
	}
	if !strings.Contains(res.Content, WorkspaceRoleQAEngineer) {
		t.Fatalf("error should list canonical role ids, got %q", res.Content)
	}
}

func TestChangeRoleRequiresExplicitTargetAndExecuteDirectory(t *testing.T) {
	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		FullCodePath: "/system/x_world/vote",
		Args: map[string]interface{}{
			"target_role": WorkspaceRoleAppOperator,
		},
	})
	if !res.IsError || !strings.Contains(res.Content, "execute_directory") {
		t.Fatalf("expected missing execute_directory to fail, got %#v", res)
	}

	res = (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		FullCodePath: "/system/x_world/vote",
		Args: map[string]interface{}{
			"execute_directory": "/system/x_world/vote",
		},
	})
	if !res.IsError || !strings.Contains(res.Content, "target_role") {
		t.Fatalf("expected missing target_role to fail, got %#v", res)
	}
}

func TestChangeRoleUsesScheduledAgentWorkspaceRootWhenDirectoryMissing(t *testing.T) {
	ctx := contextWithScheduledAgentWorkspaceRoot(context.Background(), "/system/test22/hot_news")
	res := (&ChangeRoleTool{}).Execute(ctx, ToolCall{
		FullCodePath: "/system/test22/hot_news",
		Args: map[string]interface{}{
			"target_role": WorkspaceRoleAppOperator,
		},
	})
	if res.IsError {
		t.Fatalf("scheduled change_role should fill execute_directory from task root, got %q", res.Content)
	}
	data, ok := res.Data.(changeRoleData)
	if !ok {
		t.Fatalf("expected changeRoleData, got %T", res.Data)
	}
	if data.ExecuteDirectory != "/system/test22/hot_news" || data.Handoff.ExecuteDirectory != "/system/test22/hot_news" {
		t.Fatalf("execute directory should be task root, got %#v", data.Handoff)
	}
}

func TestBuildChangeRoleLocksScheduledAgentDirectoryToTaskRoot(t *testing.T) {
	ctx := contextWithScheduledAgentWorkspaceRoot(context.Background(), "/system/test22/hot_news")
	got := buildChangeRole(ctx, changeRoleArgs{
		TargetRole:       WorkspaceRoleAppOperator,
		ExecuteDirectory: "/system/hot_news",
		TaskContext:      []string{"定时执行热点情报推送"},
	}, "/system/test22/hot_news")

	if got.ExecuteDirectory != "/system/test22/hot_news" || got.Handoff.ExecuteDirectory != "/system/test22/hot_news" {
		t.Fatalf("scheduled handoff should be locked to task root, got %#v", got.Handoff)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	if !strings.Contains(keyInfo, "模型请求切换到 /system/hot_news") ||
		!strings.Contains(keyInfo, "任务绑定目录是 /system/test22/hot_news") {
		t.Fatalf("handoff should explain directory guard, got %#v", got.Handoff.KeyInformation)
	}
}

func TestChangeRoleNormalizesNewAppTargetToSelectedParentDirectory(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole:       WorkspaceRoleAppDeveloper,
		ExecuteDirectory: "/system/ticket_sys/v1/ticket",
		TaskContext:      []string{"已确认 PRD，开发工单管理系统"},
		KeyInformation:   []string{"目标应用目录：/system/ticket_sys/v1/ticket"},
	}, "/system/ticket_sys/v1")

	if got.ExecuteDirectory != "/system/ticket_sys/v1" {
		t.Fatalf("execute directory = %q, want /system/ticket_sys/v1", got.ExecuteDirectory)
	}
	if !containsWorkspaceRoleString(got.Handoff.KeyInformation, "新建应用目标目录：/system/ticket_sys/v1/ticket") ||
		!containsWorkspaceRoleString(got.Handoff.KeyInformation, "开发阶段先在已存在父目录 /system/ticket_sys/v1 下创建目标目录，再在目标目录写代码。") {
		t.Fatalf("handoff should preserve selected parent and target child, got %#v", got.Handoff.KeyInformation)
	}
}
