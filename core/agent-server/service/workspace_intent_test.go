package service

import "testing"

func TestWorkspaceIntentSpecPlan(t *testing.T) {
	got := workspaceIntentSpec("app.plan")
	if got.ID != "app.plan" {
		t.Fatalf("spec ID=%s want app.plan", got.ID)
	}
	if !containsIntentString(got.Docs, "/system/prompt/intents/app-plan") {
		t.Fatalf("app.plan should require plan SOP, docs=%v", got.Docs)
	}
	if !containsIntentString(got.NextTools, "write_prd") {
		t.Fatalf("app.plan should allow write_prd, tools=%v", got.NextTools)
	}
}

func TestWorkspaceIntentSpecCreate(t *testing.T) {
	got := workspaceIntentSpec("app.create")
	if got.ID != "app.create" {
		t.Fatalf("spec ID=%s want app.create", got.ID)
	}
	if !containsIntentString(got.Docs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("app.create should require full SDK readme, docs=%v", got.Docs)
	}
	if containsIntentString(got.NextTools, "write_prd") {
		t.Fatalf("app.create should not output PRD again, tools=%v", got.NextTools)
	}
}

func TestWorkspaceIntentSpecBuildFix(t *testing.T) {
	got := workspaceIntentSpec("app.build_fix")
	if got.ID != "app.build_fix" {
		t.Fatalf("spec ID=%s want app.build_fix", got.ID)
	}
	if !containsIntentString(got.Docs, "/system/prompt/intents/app-build-fix") {
		t.Fatalf("build fix should include build-fix SOP, docs=%v", got.Docs)
	}
	if containsIntentString(got.Docs, "/system/prompt/sdk/build-validation-reference") {
		t.Fatalf("build fix should not auto-inject retired build reference, docs=%v", got.Docs)
	}
}

func TestWorkspaceIntentSpecUnknownFallsBackToExplainReview(t *testing.T) {
	got := workspaceIntentSpec("unknown.role")
	if got.ID != "app.explain_review" {
		t.Fatalf("unknown role spec ID=%s want app.explain_review", got.ID)
	}
}

func containsIntentString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
