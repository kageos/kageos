package service

import (
	"path/filepath"
	"testing"
)

func TestRuntimeAppPathsBuildExpectedLocations(t *testing.T) {
	t.Parallel()

	paths := newRuntimeAppPaths("namespace", "alice", "demo")

	if got := paths.AppDir(); got != filepath.Join("namespace", "alice", "demo") {
		t.Fatalf("AppDir() = %s", got)
	}
	if got := paths.AppName(); got != "alice_demo" {
		t.Fatalf("AppName() = %s", got)
	}
	if got := paths.APIDir(); got != filepath.Join("namespace", "alice", "demo", "code", "api") {
		t.Fatalf("APIDir() = %s", got)
	}
	if got := paths.WorkplaceSubDir("uploads"); got != filepath.Join("namespace", "alice", "demo", "workplace", "uploads") {
		t.Fatalf("WorkplaceSubDir() = %s", got)
	}
	if got := paths.MainGoPath(); got != filepath.Join("namespace", "alice", "demo", "code", "cmd", "app", "main.go") {
		t.Fatalf("MainGoPath() = %s", got)
	}
	if got := paths.VersionJSONPath(); got != filepath.Join("namespace", "alice", "demo", "workplace", "metadata", "version.json") {
		t.Fatalf("VersionJSONPath() = %s", got)
	}
	if got := paths.CurrentAppPath(); got != filepath.Join("namespace", "alice", "demo", "workplace", "metadata", "current_app.txt") {
		t.Fatalf("CurrentAppPath() = %s", got)
	}
	if got := paths.LogFileName("v7"); got != "alice_demo_v7.log" {
		t.Fatalf("LogFileName() = %s", got)
	}
	if got := paths.LogFile("v7"); got != filepath.Join("namespace", "alice", "demo", "workplace", "logs", "alice_demo_v7.log") {
		t.Fatalf("LogFile() = %s", got)
	}
}

func TestRuntimeAppPathsNamespaceAPIImport(t *testing.T) {
	t.Parallel()

	paths := newRuntimeAppPaths("namespace", "alice", "demo")
	got := paths.NamespaceAPIImport("/ticket_system/order")
	want := "github.com/kageos/kageos/namespace/alice/demo/code/api/ticket_system/order"
	if got != want {
		t.Fatalf("NamespaceAPIImport() = %s, want %s", got, want)
	}
}

func TestRuntimeAppPathsTrimAppPrefix(t *testing.T) {
	t.Parallel()

	paths := newRuntimeAppPaths("namespace", "alice", "demo")
	got := paths.TrimAppPrefix("/alice/demo/ticket_system/order")
	if got != "ticket_system/order" {
		t.Fatalf("TrimAppPrefix() = %s", got)
	}
}
