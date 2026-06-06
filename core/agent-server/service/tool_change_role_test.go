package service

import (
	"context"
	"strings"
	"testing"
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
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_prd") {
		t.Fatalf("app_developer should not plan PRD again, tools=%v", got.AllowedNextTools)
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
	if !strings.Contains(got.ContextPolicy, "执行目录固定为 /liubeiluo/nps") {
		t.Fatalf("context policy should pin execute directory, got %q", got.ContextPolicy)
	}
	if len(got.Handoff.TaskContext) != 3 || !strings.Contains(strings.Join(got.Handoff.TaskContext, "；"), "趋势图") {
		t.Fatalf("handoff task context not preserved: %#v", got.Handoff)
	}
	if !containsWorkspaceRoleString(got.Handoff.KeyInformation, "构建版本 v4") || !containsWorkspaceRoleString(got.Handoff.References, "nps_submit.go") {
		t.Fatalf("handoff key info/references not preserved: %#v", got.Handoff)
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
	if !strings.Contains(got.ContextPolicy, "执行目录固定为 /system/x_world") {
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
		"search_tools(directory=execute_directory)",
		"主数据/配置表 -> Form 提交 -> 目标记录表 -> Chart/结果查询",
		"参数、数据、schema、业务 bug 或构建问题",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("QA handoff advice should contain %q, got %#v", want, got.Handoff.KeyInformation)
		}
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
	for _, tool := range []string{"read_dir", "read_go_file", "read_go_file_lines", "read_app_log", "search_tools"} {
		if !containsWorkspaceRoleString(got.AllowedNextTools, tool) {
			t.Fatalf("platform engineer should report base read-only tool %s, tools=%v", tool, got.AllowedNextTools)
		}
	}
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_go_file") {
		t.Fatalf("platform engineer should not report write_go_file, tools=%v", got.AllowedNextTools)
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
