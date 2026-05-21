package service

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/kageos/kageos/pkg/config"
)

func TestUpdateMainFileImportsAddsBlankImportAndIsIdempotent(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newPackageScaffoldTestService(basePath)
	mainFilePath := writeMainGoFixture(t, basePath, `package main

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	if err := service.updateMainFileImports(context.Background(), "alice", "demo", "/ticket_system/order"); err != nil {
		t.Fatalf("updateMainFileImports first call: %v", err)
	}
	if err := service.updateMainFileImports(context.Background(), "alice", "demo", "ticket_system/order"); err != nil {
		t.Fatalf("updateMainFileImports second call: %v", err)
	}

	content := readMainGoFixture(t, mainFilePath)
	importPath := `github.com/kageos/kageos/namespace/alice/demo/code/api/ticket_system/order`
	if strings.Count(content, importPath) != 1 {
		t.Fatalf("unexpected import count in main.go: %s", content)
	}

	assertValidGoFile(t, mainFilePath)
}

func TestRemoveMainFileImportRemovesOnlyTargetImport(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newPackageScaffoldTestService(basePath)
	mainFilePath := writeMainGoFixture(t, basePath, `package main

import (
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/keep/me"
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/remove/me"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	if err := service.removeMainFileImport(context.Background(), "alice", "demo", "/remove/me"); err != nil {
		t.Fatalf("removeMainFileImport: %v", err)
	}

	content := readMainGoFixture(t, mainFilePath)
	if strings.Contains(content, `github.com/kageos/kageos/namespace/alice/demo/code/api/remove/me`) {
		t.Fatalf("target import still exists: %s", content)
	}
	if !strings.Contains(content, `github.com/kageos/kageos/namespace/alice/demo/code/api/keep/me`) {
		t.Fatalf("non-target import was removed: %s", content)
	}

	assertValidGoFile(t, mainFilePath)
}

func TestRemoveMainFileImportRemovesTargetSubtreeImports(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newPackageScaffoldTestService(basePath)
	mainFilePath := writeMainGoFixture(t, basePath, `package main

import (
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/keep/me"
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/workspace"
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/workspace/create-project"
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/workspace/execute"
	_ "github.com/kageos/kageos/namespace/alice/demo/code/api/workspace_extra/keep"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	if err := service.removeMainFileImport(context.Background(), "alice", "demo", "/workspace"); err != nil {
		t.Fatalf("removeMainFileImport: %v", err)
	}

	content := readMainGoFixture(t, mainFilePath)
	for _, removed := range []string{
		`github.com/kageos/kageos/namespace/alice/demo/code/api/workspace"`,
		`github.com/kageos/kageos/namespace/alice/demo/code/api/workspace/create-project`,
		`github.com/kageos/kageos/namespace/alice/demo/code/api/workspace/execute`,
	} {
		if strings.Contains(content, removed) {
			t.Fatalf("target subtree import still exists %s: %s", removed, content)
		}
	}
	for _, kept := range []string{
		`github.com/kageos/kageos/namespace/alice/demo/code/api/keep/me`,
		`github.com/kageos/kageos/namespace/alice/demo/code/api/workspace_extra/keep`,
	} {
		if !strings.Contains(content, kept) {
			t.Fatalf("non-target import was removed %s: %s", kept, content)
		}
	}

	assertValidGoFile(t, mainFilePath)
}

func newServiceTreeTestService(basePath string) *WorkspaceChangeService {
	cfg := &appconfig.AppManageServiceConfig{
		AppDir: appconfig.AppDirConfig{
			BasePath: basePath,
		},
	}
	return NewWorkspaceChangeService(cfg, nil, NewWorkspaceFileService(cfg))
}

func newServiceTreeTestServiceWithAppManage(basePath string) *WorkspaceChangeService {
	cfg := &appconfig.AppManageServiceConfig{
		AppDir: appconfig.AppDirConfig{
			BasePath: basePath,
		},
	}
	return NewWorkspaceChangeService(cfg, &AppManageService{config: cfg}, NewWorkspaceFileService(cfg))
}

func newPackageScaffoldTestService(basePath string) *PackageScaffoldService {
	return NewPackageScaffoldService(&appconfig.AppManageServiceConfig{
		AppDir: appconfig.AppDirConfig{
			BasePath: basePath,
		},
	})
}

func newWorkspaceFileTestService(basePath string) *WorkspaceFileService {
	return NewWorkspaceFileService(&appconfig.AppManageServiceConfig{
		AppDir: appconfig.AppDirConfig{
			BasePath: basePath,
		},
	})
}

func writeMainGoFixture(t *testing.T, basePath string, content string) string {
	t.Helper()

	mainFilePath := newRuntimeAppPaths(basePath, "alice", "demo").MainGoPath()
	if err := os.MkdirAll(filepath.Dir(mainFilePath), 0755); err != nil {
		t.Fatalf("mkdir main.go dir: %v", err)
	}
	if err := os.WriteFile(mainFilePath, []byte(content), 0644); err != nil {
		t.Fatalf("write main.go fixture: %v", err)
	}
	return mainFilePath
}

func readMainGoFixture(t *testing.T, mainFilePath string) string {
	t.Helper()

	data, err := os.ReadFile(mainFilePath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	return string(data)
}

func assertValidGoFile(t *testing.T, filePath string) {
	t.Helper()

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments); err != nil {
		t.Fatalf("parse go file %s: %v", filePath, err)
	}
}
