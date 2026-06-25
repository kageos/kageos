package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/sdkmodule"
	"golang.org/x/mod/modfile"
)

func TestCreateVersionFilesAlsoCreatesCurrentVersionFiles(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)

	if err := service.createVersionFiles("alice", "demo"); err != nil {
		t.Fatalf("createVersionFiles: %v", err)
	}

	paths := service.getVersionMetadataPaths("alice", "demo")
	assertFileContent(t, paths.currentVersionPath, "v1")
	assertFileContent(t, paths.currentAppPath, "alice_demo")
	if _, err := os.Stat(paths.versionJSONPath); err != nil {
		t.Fatalf("expected version.json to exist: %v", err)
	}
}

func TestUpdateCurrentVersionFilesUsesConfiguredBasePath(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)

	if err := service.updateCurrentVersionFiles("alice", "demo", "v9"); err != nil {
		t.Fatalf("updateCurrentVersionFiles: %v", err)
	}

	paths := service.getVersionMetadataPaths("alice", "demo")
	assertFileContent(t, paths.currentVersionPath, "v9")
	assertFileContent(t, paths.currentAppPath, "alice_demo")

	hardcodedNamespacePath := filepath.Join("namespace", "alice", "demo", "workplace", "metadata", "current_version.txt")
	if _, err := os.Stat(hardcodedNamespacePath); err == nil {
		t.Fatalf("unexpected write to hardcoded namespace path: %s", hardcodedNamespacePath)
	}
}

func TestUpdateVersionJSONUsesConfiguredBasePathForCurrentVersionFiles(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appDir := filepath.Join(basePath, "alice", "demo")
	metadataDir := filepath.Join(appDir, "workplace", "metadata")

	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}

	if err := service.createVersionFiles("alice", "demo"); err != nil {
		t.Fatalf("createVersionFiles: %v", err)
	}

	if err := service.updateVersionJson(appDir, "alice", "demo", "v2"); err != nil {
		t.Fatalf("updateVersionJson: %v", err)
	}

	paths := service.getVersionMetadataPaths("alice", "demo")
	assertFileContent(t, paths.currentVersionPath, "v2")

	versionData, err := service.readVersionData(paths.versionJSONPath)
	if err != nil {
		t.Fatalf("readVersionData: %v", err)
	}
	if versionData.CurrentVersion != "v2" || versionData.LatestVersion != "v2" {
		t.Fatalf("unexpected version data: %+v", versionData)
	}
}

func TestWriteBuiltRuntimeManifestCreatesObservableFile(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")

	if err := service.writeBuiltRuntimeManifest("alice", "demo", appPaths, "v7"); err != nil {
		t.Fatalf("writeBuiltRuntimeManifest: %v", err)
	}

	manifest, err := service.readRuntimeManifest(appPaths.RuntimeManifestPath())
	if err != nil {
		t.Fatalf("readRuntimeManifest: %v", err)
	}
	if manifest.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", manifest.SchemaVersion)
	}
	if manifest.User != "alice" || manifest.App != "demo" || manifest.Version != "v7" {
		t.Fatalf("unexpected identity: %+v", manifest)
	}
	if manifest.VersionNum != 7 {
		t.Fatalf("VersionNum = %d, want 7", manifest.VersionNum)
	}
	if manifest.BinaryName != "alice_demo_v7" {
		t.Fatalf("BinaryName = %q, want alice_demo_v7", manifest.BinaryName)
	}
	if manifest.BinaryPath != "/app/workplace/bin/releases/alice_demo_v7" {
		t.Fatalf("BinaryPath = %q, want container binary path", manifest.BinaryPath)
	}
	if manifest.HostBinaryPath != filepath.Join(basePath, "alice", "demo", "workplace", "bin", "releases", "alice_demo_v7") {
		t.Fatalf("HostBinaryPath = %q", manifest.HostBinaryPath)
	}
	if manifest.Status != "built" {
		t.Fatalf("Status = %q, want built", manifest.Status)
	}
	if manifest.CreatedAt == "" || manifest.BuiltAt == "" || manifest.UpdatedAt == "" {
		t.Fatalf("expected timestamps to be populated: %+v", manifest)
	}
}

func TestUpdateRuntimeManifestStartupPreservesBuildInfo(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	builtAt := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	manifest := service.buildRuntimeManifest("alice", "demo", appPaths, "v2", builtAt)
	if err := os.MkdirAll(appPaths.MetadataDir(), 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}
	if err := service.writeRuntimeManifest(appPaths.RuntimeManifestPath(), &manifest); err != nil {
		t.Fatalf("writeRuntimeManifest: %v", err)
	}

	startedAt := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	if err := service.updateRuntimeManifestStartup(&StartupNotification{
		User:      "alice",
		App:       "demo",
		Version:   "v2",
		Status:    "started",
		StartTime: startedAt,
	}); err != nil {
		t.Fatalf("updateRuntimeManifestStartup: %v", err)
	}

	got, err := service.readRuntimeManifest(appPaths.RuntimeManifestPath())
	if err != nil {
		t.Fatalf("readRuntimeManifest: %v", err)
	}
	if got.Status != "running" {
		t.Fatalf("Status = %q, want running", got.Status)
	}
	if got.StartedAt != startedAt.Format(time.RFC3339) {
		t.Fatalf("StartedAt = %q, want %q", got.StartedAt, startedAt.Format(time.RFC3339))
	}
	if got.BuiltAt != builtAt.Format(time.RFC3339) || got.CreatedAt != builtAt.Format(time.RFC3339) {
		t.Fatalf("expected build timestamps to be preserved: %+v", got)
	}
}

func TestUpdateRuntimeManifestStartupSkipsDifferentVersion(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	manifest := service.buildRuntimeManifest("alice", "demo", appPaths, "v3", time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC))
	if err := os.MkdirAll(appPaths.MetadataDir(), 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}
	if err := service.writeRuntimeManifest(appPaths.RuntimeManifestPath(), &manifest); err != nil {
		t.Fatalf("writeRuntimeManifest: %v", err)
	}

	if err := service.updateRuntimeManifestStartup(&StartupNotification{
		User:      "alice",
		App:       "demo",
		Version:   "v2",
		Status:    "running",
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("updateRuntimeManifestStartup: %v", err)
	}

	got, err := service.readRuntimeManifest(appPaths.RuntimeManifestPath())
	if err != nil {
		t.Fatalf("readRuntimeManifest: %v", err)
	}
	if got.Version != "v3" || got.Status != "built" || got.StartedAt != "" {
		t.Fatalf("expected different-version startup to be ignored: %+v", got)
	}
}

func TestEnsureAppGoModFileCreatesWithCurrentSDK(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	if err := os.MkdirAll(appPaths.AppDir(), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	if err := service.ensureAppGoModFile(appPaths); err != nil {
		t.Fatalf("ensureAppGoModFile: %v", err)
	}

	if got := readGoModRequireVersion(t, appPaths.GoModPath(), sdkmodule.ModulePath); got != sdkmodule.Version {
		t.Fatalf("SDK version = %q, want %q", got, sdkmodule.Version)
	}
}

func TestEnsureAppGoModFileUpgradesOlderSDK(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	if err := os.MkdirAll(appPaths.AppDir(), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	goMod := `module namespace/alice/demo

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/kageos/kageos-sdk v0.1.0
)
`
	if err := os.WriteFile(appPaths.GoModPath(), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := service.ensureAppGoModFile(appPaths); err != nil {
		t.Fatalf("ensureAppGoModFile: %v", err)
	}

	if got := readGoModRequireVersion(t, appPaths.GoModPath(), sdkmodule.ModulePath); got != sdkmodule.Version {
		t.Fatalf("SDK version = %q, want %q", got, sdkmodule.Version)
	}
	if got := readGoModRequireVersion(t, appPaths.GoModPath(), "github.com/google/uuid"); got != "v1.6.0" {
		t.Fatalf("uuid version = %q, want v1.6.0", got)
	}
}

func TestEnsureAppGoModFileDoesNotDowngradeNewerSDK(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newVersionFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	if err := os.MkdirAll(appPaths.AppDir(), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	goMod := `module namespace/alice/demo

go 1.25.0

require github.com/kageos/kageos-sdk v0.99.0
`
	if err := os.WriteFile(appPaths.GoModPath(), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := service.ensureAppGoModFile(appPaths); err != nil {
		t.Fatalf("ensureAppGoModFile: %v", err)
	}

	if got := readGoModRequireVersion(t, appPaths.GoModPath(), sdkmodule.ModulePath); got != "v0.99.0" {
		t.Fatalf("SDK version = %q, want v0.99.0", got)
	}
}

func newVersionFileTestService(basePath string) *AppManageService {
	return &AppManageService{
		config: &appconfig.AppManageServiceConfig{
			AppDir: appconfig.AppDirConfig{
				BasePath: basePath,
			},
		},
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("unexpected file content for %s: got %q want %q", path, string(data), want)
	}
}

func readGoModRequireVersion(t *testing.T, path, module string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	file, err := modfile.Parse(path, data, nil)
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	for _, req := range file.Require {
		if req.Mod.Path == module {
			return req.Mod.Version
		}
	}
	t.Fatalf("module %s not found in %s", module, path)
	return ""
}
