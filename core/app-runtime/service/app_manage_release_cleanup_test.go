package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupReleaseBinariesKeepsCurrentAndLatestVersions(t *testing.T) {
	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	releasesDir := appPaths.BuildOutputDir(service.config.GetBuildOutputDir())
	if err := os.MkdirAll(releasesDir, 0755); err != nil {
		t.Fatalf("mkdir releases: %v", err)
	}
	if err := os.MkdirAll(appPaths.LogsDir(), 0755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	if err := os.MkdirAll(appPaths.MetadataDir(), 0755); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}

	versionData := &VersionData{
		User:           "alice",
		App:            "demo",
		CurrentVersion: "v2",
		LatestVersion:  "v5",
		Versions: []VersionInfo{
			{Version: "v1", CreatedAt: time.Now().Add(-5 * time.Hour).Format(time.RFC3339), Status: "inactive"},
			{Version: "v2", CreatedAt: time.Now().Add(-4 * time.Hour).Format(time.RFC3339), Status: "active"},
			{Version: "v3", CreatedAt: time.Now().Add(-3 * time.Hour).Format(time.RFC3339), Status: "inactive"},
			{Version: "v4", CreatedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339), Status: "inactive"},
			{Version: "v5", CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339), Status: "inactive"},
		},
	}
	if err := service.writeVersionData(appPaths.VersionJSONPath(), versionData); err != nil {
		t.Fatalf("write version data: %v", err)
	}

	for _, version := range []string{"v1", "v2", "v3", "v4", "v5"} {
		writeReleaseBinary(t, releasesDir, service.appBinaryName("alice", "demo", version))
		writeReleaseBinary(t, appPaths.LogsDir(), appPaths.LogFileName(version))
	}
	writeReleaseBinary(t, appPaths.LogsDir(), "alice_demo_v1-2026-07-05T00-00-00.000.log.gz")
	writeReleaseBinary(t, appPaths.LogsDir(), "alice_demo_v2-2026-07-05T00-00-00.000.log.gz")
	writeReleaseBinary(t, releasesDir, "manual-note.txt")
	writeReleaseBinary(t, appPaths.LogsDir(), "manual-debug.log")
	if err := os.Mkdir(filepath.Join(releasesDir, "subdir"), 0755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	stats, err := service.cleanupReleaseBinariesForApp(context.Background(), "alice", "demo", 3)
	if err != nil {
		t.Fatalf("cleanupReleaseBinariesForApp: %v", err)
	}
	if stats.removed != 3 {
		t.Fatalf("removed = %d, want 3", stats.removed)
	}

	assertPathMissing(t, filepath.Join(releasesDir, "alice_demo_v1"))
	assertPathMissing(t, filepath.Join(appPaths.LogsDir(), "alice_demo_v1.log"))
	assertPathMissing(t, filepath.Join(appPaths.LogsDir(), "alice_demo_v1-2026-07-05T00-00-00.000.log.gz"))
	for _, name := range []string{"alice_demo_v2", "alice_demo_v3", "alice_demo_v4", "alice_demo_v5", "manual-note.txt"} {
		assertPathExists(t, filepath.Join(releasesDir, name))
	}
	for _, name := range []string{"alice_demo_v2.log", "alice_demo_v2-2026-07-05T00-00-00.000.log.gz", "alice_demo_v3.log", "alice_demo_v4.log", "alice_demo_v5.log", "manual-debug.log"} {
		assertPathExists(t, filepath.Join(appPaths.LogsDir(), name))
	}
}

func TestReleaseVersionsToKeepIncludesCurrentEvenWhenOld(t *testing.T) {
	versionData := &VersionData{
		CurrentVersion: "v1",
		LatestVersion:  "v5",
		Versions: []VersionInfo{
			{Version: "v1"},
			{Version: "v2"},
			{Version: "v3"},
			{Version: "v4"},
		},
	}

	keep := releaseVersionsToKeep(versionData, 2)
	for _, version := range []string{"v1", "v5", "v4", "v3"} {
		if _, ok := keep[version]; !ok {
			t.Fatalf("expected %s to be kept, keep=%v", version, keep)
		}
	}
	if _, ok := keep["v2"]; ok {
		t.Fatalf("did not expect v2 to be kept, keep=%v", keep)
	}
}

func writeReleaseBinary(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("binary"), 0755); err != nil {
		t.Fatalf("write release binary %s: %v", name, err)
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist %s: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected path to be missing: %s", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
