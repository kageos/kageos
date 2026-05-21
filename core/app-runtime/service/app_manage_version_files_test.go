package service

import (
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/kageos/kageos/pkg/config"
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
