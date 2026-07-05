package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestResolveSourceFileWriteTargetRejectsTestFiles(t *testing.T) {
	t.Parallel()

	appPaths := newRuntimeAppPaths(t.TempDir(), "luobei", "demo")
	for _, fileName := range []string{"ticket_test.go", "ticket_test"} {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveSourceFileWriteTarget(appPaths, &dto.SourceFileWrite{
				DirectoryPath: "tickets",
				FileName:      fileName,
				SourceCode:    "package tickets\n",
			})
			if err == nil {
				t.Fatalf("expected _test.go source file to fail")
			}
			if !strings.Contains(err.Error(), "_test.go") || !strings.Contains(err.Error(), "API 注册") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveSourceFileWriteTargetRejectsInternalManifestFiles(t *testing.T) {
	t.Parallel()

	appPaths := newRuntimeAppPaths(t.TempDir(), "luobei", "demo")
	for _, fileName := range []string{"kageos_manifest.go", "kageos_manifest"} {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveSourceFileWriteTarget(appPaths, &dto.SourceFileWrite{
				DirectoryPath: "tickets",
				FileName:      fileName,
				SourceCode:    "package tickets\n",
			})
			if err == nil {
				t.Fatalf("expected internal manifest source file to fail")
			}
			if !strings.Contains(err.Error(), "本地目录种子声明") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveWorkspaceFilePathRejectsInternalManifestFiles(t *testing.T) {
	t.Parallel()

	appPaths := newRuntimeAppPaths(t.TempDir(), "luobei", "demo")
	for _, fileName := range []string{"kageos_manifest.go", "kageos_manifest"} {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			_, err := resolveWorkspaceFilePath(appPaths, "tickets", fileName)
			if err == nil {
				t.Fatalf("expected internal manifest workspace file to fail")
			}
			if !strings.Contains(err.Error(), "本地目录种子声明") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReadDirectoryFilesHidesWorkspaceInternalManifestFiles(t *testing.T) {
	basePath := t.TempDir()
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	targetDir := filepath.Join(appPaths.APIDir(), "followup")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"kageos_manifest.go": "package followup\n",
		"actions.go":         "package followup\n",
	} {
		if err := os.WriteFile(filepath.Join(targetDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := newWorkspaceFileTestService(basePath).ReadDirectoryFiles(t.Context(), "alice", "demo", "/alice/demo/followup")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].RelativePath != "actions.go" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestReadDirectoryFilesIncludesSupportedTextFiles(t *testing.T) {
	basePath := t.TempDir()
	appPaths := newRuntimeAppPaths(basePath, "alice", "demo")
	targetDir := filepath.Join(appPaths.APIDir(), "followup")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"actions.go":   "package followup\n",
		"config.json":  `{"enabled":true}`,
		"template.md":  "# Title\n",
		"preview.webp": "binary-ish",
	} {
		if err := os.WriteFile(filepath.Join(targetDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := newWorkspaceFileTestService(basePath).ReadDirectoryFiles(t.Context(), "alice", "demo", "/alice/demo/followup")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, file := range files {
		got[file.RelativePath] = file.FileType
	}
	for name, typ := range map[string]string{"actions.go": "go", "config.json": "json", "template.md": "md"} {
		if got[name] != typ {
			t.Fatalf("expected %s as %s, got files=%#v", name, typ, files)
		}
	}
	if _, ok := got["preview.webp"]; ok {
		t.Fatalf("binary-like file should not be returned: %#v", files)
	}
}

func TestResolveWorkspaceTextFilePathRejectsUnsupportedExtension(t *testing.T) {
	t.Parallel()

	appPaths := newRuntimeAppPaths(t.TempDir(), "luobei", "demo")
	_, _, err := resolveWorkspaceTextFilePath(appPaths, "/luobei/demo/assets", "photo.webp", "")
	if err == nil {
		t.Fatalf("expected unsupported binary extension to fail")
	}
	if !strings.Contains(err.Error(), "仅支持文本资源") {
		t.Fatalf("unexpected error: %v", err)
	}
}
