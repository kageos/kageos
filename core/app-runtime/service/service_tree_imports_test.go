package service

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
)

func TestUpdateMainFileImportsAddsBlankImportAndIsIdempotent(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)
	mainFilePath := writeMainGoFixture(t, basePath, `package main

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
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
	importPath := `github.com/ai-agent-os/ai-agent-os/namespace/alice/demo/code/api/ticket_system/order`
	if strings.Count(content, importPath) != 1 {
		t.Fatalf("unexpected import count in main.go: %s", content)
	}

	assertValidGoFile(t, mainFilePath)
}

func TestRemoveMainFileImportRemovesOnlyTargetImport(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)
	mainFilePath := writeMainGoFixture(t, basePath, `package main

import (
	_ "github.com/ai-agent-os/ai-agent-os/namespace/alice/demo/code/api/keep/me"
	_ "github.com/ai-agent-os/ai-agent-os/namespace/alice/demo/code/api/remove/me"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
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
	if strings.Contains(content, `github.com/ai-agent-os/ai-agent-os/namespace/alice/demo/code/api/remove/me`) {
		t.Fatalf("target import still exists: %s", content)
	}
	if !strings.Contains(content, `github.com/ai-agent-os/ai-agent-os/namespace/alice/demo/code/api/keep/me`) {
		t.Fatalf("non-target import was removed: %s", content)
	}

	assertValidGoFile(t, mainFilePath)
}

func newServiceTreeTestService(basePath string) *ServiceTreeService {
	return &ServiceTreeService{
		config: &appconfig.AppManageServiceConfig{
			AppDir: appconfig.AppDirConfig{
				BasePath: basePath,
			},
		},
	}
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
