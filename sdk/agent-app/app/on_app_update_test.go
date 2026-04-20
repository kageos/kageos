package app

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
)

func TestCollectPackageInfosSortsByDepthThenPath(t *testing.T) {
	oldUser, oldApp := env.User, env.App
	env.User, env.App = "alice", "demo"
	defer func() {
		env.User, env.App = oldUser, oldApp
	}()

	app := &App{
		routerInfo: map[string]*routerInfo{
			"/zeta/run": {
				Options: &RegisterOptions{PackagePath: "zeta"},
			},
			"/alpha/beta/run": {
				Options: &RegisterOptions{PackagePath: "alpha/beta"},
			},
			"/alpha/gamma/run": {
				Options: &RegisterOptions{PackagePath: "alpha/gamma"},
			},
		},
		packageContexts: map[string]*PackageContext{
			"alpha": {Name: "Alpha", Desc: `alpha package`},
		},
	}

	got := app.collectPackageInfos()
	wantPaths := []string{
		"/alice/demo/alpha",
		"/alice/demo/zeta",
		"/alice/demo/alpha/beta",
		"/alice/demo/alpha/gamma",
	}
	if len(got) != len(wantPaths) {
		t.Fatalf("expected %d packages, got %d: %#v", len(wantPaths), len(got), got)
	}

	for i, want := range wantPaths {
		if got[i].FullPath != want {
			t.Fatalf("package %d: want %s, got %s", i, want, got[i].FullPath)
		}
	}
	if got[0].Name != "Alpha" || got[0].Desc != "alpha package" {
		t.Fatalf("expected package context metadata to be applied, got %+v", got[0])
	}
}

func TestSortApiInfosByKey(t *testing.T) {
	apis := []*ApiInfo{
		{Method: "POST", Router: "/ticket/create"},
		{Method: "GET", Router: "/ticket/list"},
		{Method: "DELETE", Router: "/ticket/delete"},
	}

	sortApiInfosByKey(apis)

	want := []string{
		"DELETE:/ticket/delete",
		"GET:/ticket/list",
		"POST:/ticket/create",
	}
	for i, api := range apis {
		got := api.Method + ":" + api.Router
		if got != want[i] {
			t.Fatalf("api %d: want %s, got %s", i, want[i], got)
		}
	}
}
