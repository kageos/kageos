package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanRuntimeCleanupsKeepsCurrentLatestAndRecent(t *testing.T) {
	namespaceRoot := t.TempDir()
	appDir := filepath.Join(namespaceRoot, "alice", "demo")
	metadataDir := filepath.Join(appDir, "workplace", "metadata")
	releasesDir := filepath.Join(appDir, "workplace", "bin", "releases")
	logsDir := filepath.Join(appDir, "workplace", "logs")

	writeCleanTestFile(t, filepath.Join(metadataDir, "version.json"), `{
  "user": "alice",
  "app": "demo",
  "current_version": "v2",
  "latest_version": "v5",
  "versions": [
    {"version": "v1"},
    {"version": "v2"},
    {"version": "v3"},
    {"version": "v4"},
    {"version": "v5"}
  ]
}`)
	for _, version := range []string{"v1", "v2", "v3", "v4", "v5"} {
		writeCleanTestFile(t, filepath.Join(releasesDir, "alice_demo_"+version), "binary")
		writeCleanTestFile(t, filepath.Join(logsDir, "alice_demo_"+version+".log"), "log")
		writeCleanTestFile(t, filepath.Join(logsDir, "alice_demo_"+version+"-2026-07-05T00-00-00.000.log.gz"), "rotated")
	}
	writeCleanTestFile(t, filepath.Join(releasesDir, "manual-note.txt"), "keep")
	writeCleanTestFile(t, filepath.Join(logsDir, "manual-debug.log"), "keep")

	targets, err := planRuntimeCleanups(namespaceRoot, 3)
	if err != nil {
		t.Fatal(err)
	}

	got := cleanTargetBasenames(targets)
	for _, want := range []string{
		"alice_demo_v1",
		"alice_demo_v1.log",
		"alice_demo_v1-2026-07-05T00-00-00.000.log.gz",
	} {
		if !got[want] {
			t.Fatalf("expected cleanup target %s, got %#v", want, got)
		}
	}
	for _, keep := range []string{"alice_demo_v2", "alice_demo_v3", "alice_demo_v4", "alice_demo_v5", "manual-note.txt", "manual-debug.log"} {
		if got[keep] {
			t.Fatalf("did not expect cleanup target %s, got %#v", keep, got)
		}
	}
}

func TestPlanSourceLogCleanupsSkipsRuntimeDataAndRootLogs(t *testing.T) {
	repoRoot := t.TempDir()
	namespaceRoot := filepath.Join(repoRoot, "namespace")

	include := []string{
		"core/app-runtime/service/logs/app.log",
		"pkg/auth/logs/app.log",
		"sdk/agent-app/runtime/python/logs/app.log",
		"web/.npm-cache/_logs/2026-07-05-debug-0.log",
		"namespace/system/demo/code/api/example/logs/app.log",
		"namespace/system/demo/logs/app.log",
	}
	for _, rel := range include {
		writeCleanTestFile(t, filepath.Join(repoRoot, rel), "log")
	}

	skip := []string{
		"logs/all-services.log",
		"namespace/system/demo/workplace/logs/system_demo_v1.log",
		"namespace/system/demo/workplace/data/runtime.log",
	}
	for _, rel := range skip {
		writeCleanTestFile(t, filepath.Join(repoRoot, rel), "keep")
	}

	targets, err := planSourceLogCleanups(repoRoot, namespaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	got := cleanTargetRelPaths(t, repoRoot, targets)
	for _, want := range include {
		if !got[want] {
			t.Fatalf("expected source log target %s, got %#v", want, got)
		}
	}
	for _, keep := range skip {
		if got[keep] {
			t.Fatalf("did not expect source log target %s, got %#v", keep, got)
		}
	}
}

func TestPlanSourceLogCleanupsIncludesEmptyLogDirs(t *testing.T) {
	repoRoot := t.TempDir()
	namespaceRoot := filepath.Join(repoRoot, "namespace")

	emptyDirs := []string{
		".kageos/local-verify-c178-data/logs",
		"core/app-runtime/service/logs",
		"pkg/auth/logs",
		"namespace/system/demo/code/api/example/logs",
		"namespace/system/demo/logs",
	}
	for _, rel := range emptyDirs {
		if err := os.MkdirAll(filepath.Join(repoRoot, rel), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeCleanTestFile(t, filepath.Join(repoRoot, "namespace/system/demo/workplace/logs/system_demo_v1.log"), "keep")

	targets, err := planSourceLogCleanups(repoRoot, namespaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	got := cleanTargetRelPaths(t, repoRoot, targets)
	for _, want := range emptyDirs {
		if !got[want] {
			t.Fatalf("expected empty log dir target %s, got %#v", want, got)
		}
	}
	if got["namespace/system/demo/workplace/logs"] {
		t.Fatalf("did not expect non-empty workplace log dir target, got %#v", got)
	}
}

func TestParseCleanFlags(t *testing.T) {
	opts, err := parseCleanFlags([]string{"runtime", "--keep", "2", "--execute"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Target != "runtime" || opts.Keep != 2 || !opts.Execute {
		t.Fatalf("unexpected clean opts: %#v", opts)
	}

	if _, err := parseCleanFlags([]string{"logs", "--keep", "2"}); err == nil || !strings.Contains(err.Error(), "does not support --keep") {
		t.Fatalf("expected clean logs --keep error, got %v", err)
	}
}

func writeCleanTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func cleanTargetBasenames(targets []cleanTarget) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[filepath.Base(target.Path)] = true
	}
	return out
}

func cleanTargetRelPaths(t *testing.T, root string, targets []cleanTarget) map[string]bool {
	t.Helper()
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		rel, err := filepath.Rel(root, target.Path)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.ToSlash(rel)] = true
	}
	return out
}
