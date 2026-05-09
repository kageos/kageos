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
	}
}

func TestNormalizeAndAliases(t *testing.T) {
	cases := map[string]string{
		"product-manager":      ProductManager,
		"app-developer":        AppDeveloper,
		"qa-engineer":          QAEngineer,
		"app-operator":         AppOperator,
		"build-engineer":       BuildEngineer,
		"maintenance-engineer": MaintenanceEngineer,
		"data-operator":        DataOperator,
		"scheduler-engineer":   SchedulerEngineer,
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
		"### `app_operator` 应用操作员",
		"`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段",
		"### `reviewer` 代码审查分析师",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("routing markdown should contain %q, got:\n%s", want, got)
		}
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
