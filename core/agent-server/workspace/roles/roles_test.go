package roles

import (
	"strings"
	"testing"
)

func TestSpecsExposeRoleContracts(t *testing.T) {
	specs := Specs()
	for _, roleID := range RouteOrder() {
		spec, ok := specs[roleID]
		if !ok {
			t.Fatalf("missing role spec %q", roleID)
		}
		if strings.TrimSpace(spec.ID) == "" || strings.TrimSpace(spec.DisplayName) == "" {
			t.Fatalf("role spec should have id and display name: %#v", spec)
		}
		if strings.TrimSpace(spec.RouteDescription) == "" {
			t.Fatalf("role spec %q should have route description", roleID)
		}
		if len(spec.Docs) == 0 {
			t.Fatalf("role spec %q should load at least one SOP doc", roleID)
		}
		if len(spec.Runtime.HandoffRequired) == 0 {
			t.Fatalf("role spec %q should declare handoff required fields", roleID)
		}
		if len(spec.Runtime.SOP) == 0 {
			t.Fatalf("role spec %q should declare runtime SOP", roleID)
		}
		if len(spec.Runtime.DoneWhen) == 0 {
			t.Fatalf("role spec %q should declare done_when", roleID)
		}
	}
}

func TestRoleRuntimeContractsExposeLifecycleHooks(t *testing.T) {
	specs := Specs()
	for _, tc := range []struct {
		roleID string
		hookID string
		stage  string
	}{
		{roleID: ProductManager, hookID: "product_manager.to_app_developer", stage: "before_handoff"},
		{roleID: AppDeveloper, hookID: "app_developer.before_enter_prd", stage: "before_enter"},
		{roleID: AppOperator, hookID: "app_operator.before_enter_capabilities", stage: "before_enter"},
		{roleID: BuildEngineer, hookID: "build_engineer.before_enter_diagnostics", stage: "before_enter"},
	} {
		spec := specs[tc.roleID]
		found := false
		for _, h := range spec.Runtime.Hooks {
			if h.ID == tc.hookID && h.Stage == tc.stage {
				found = true
				if strings.TrimSpace(h.Purpose) == "" || len(h.Produces) == 0 {
					t.Fatalf("hook %s should declare purpose and outputs: %#v", tc.hookID, h)
				}
			}
		}
		if !found {
			t.Fatalf("role %s should expose hook %s at %s, hooks=%#v", tc.roleID, tc.hookID, tc.stage, spec.Runtime.Hooks)
		}
	}
}

func TestNormalizeAndAliases(t *testing.T) {
	cases := map[string]string{
		"product-manager":      ProductManager,
		"app-developer":        AppDeveloper,
		"qa-engineer":          QAEngineer,
		"app-operator":         AppOperator,
		"automation-operator":  AutomationOperator,
		"build-engineer":       BuildEngineer,
		"maintenance-engineer": MaintenanceEngineer,
		"data-operator":        DataOperator,
		"platform-engineer":    PlatformEngineer,
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", input, got, want)
		}
	}
}

func TestRoutingMarkdownIsGeneratedFromSpecs(t *testing.T) {
	got := RoutingMarkdown()
	for _, want := range []string{
		"## 角色路由",
		"### `product_manager` 产品经理",
		"### `app_developer` 应用开发工程师",
		"### `app_operator` 应用执行",
		"### `automation_operator` 自动执行配置",
		"当前目录已是目标应用",
		"不依赖某个固定动词",
		"使用软件完成业务结果",
		"轻量一次性文件/数据任务",
		"复杂、专项或多步骤",
		"自动执行配置负责以后自动执行",
		"已有应用函数、已有业务操作或已有工作台目录",
		"维护长期数据",
		"当前目录、本空间其他目录、其他空间函数、系统工具和连接器函数",
		"预期使用工具清单",
		"按业务场景裁剪的质量控制",
		"不要把示例规则机械套到所有任务",
		"如果用户想定时执行的能力还不存在",
		"在 `/system/x_world/vote` 里“创建一个投票”是业务操作",
		"`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段",
		"### `reviewer` 代码审查分析师",
		"kageos 是什么",
		"/system/prompt/platform-introduction",
		"kageos 怎么用",
		"/system/prompt/platform-usage-and-philosophy",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("routing markdown should contain %q, got:\n%s", want, got)
		}
	}
	if strings.Index(got, "### `app_operator` 应用执行") > strings.Index(got, "### `product_manager` 产品经理") {
		t.Fatalf("app_operator should be shown before product_manager so existing-app operations are considered first:\n%s", got)
	}
}

func TestTransitionWhenUsesRoleContracts(t *testing.T) {
	when, ok := TransitionWhen(AppDeveloper, QAEngineer)
	if !ok {
		t.Fatalf("app developer should recommend QA as next role")
	}
	if !strings.Contains(when, "build 成功") {
		t.Fatalf("unexpected transition reason: %q", when)
	}
	if _, ok := TransitionWhen(ProductManager, BuildEngineer); ok {
		t.Fatalf("product manager should not directly recommend build engineer")
	}
}

func TestApplyRoutingMarkdown(t *testing.T) {
	got := ApplyRoutingMarkdown("before\n" + RoutingMarker + "\nafter")
	if strings.Contains(got, RoutingMarker) {
		t.Fatalf("routing marker should be replaced: %s", got)
	}
	if !strings.Contains(got, "### `qa_engineer` 测试工程师") {
		t.Fatalf("routing markdown not injected: %s", got)
	}
}
