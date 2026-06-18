package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestRollbackFilesRestoresOverwrittenFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ticket.go")
	originalContent := []byte("package ticket\n\nconst Name = \"old\"\n")

	if err := os.WriteFile(filePath, originalContent, 0640); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	service := newWorkspaceFileTestService(tempDir)
	entry, err := service.captureFileRollbackEntry(filePath)
	if err != nil {
		t.Fatalf("capture rollback entry: %v", err)
	}

	if err := os.WriteFile(filePath, []byte("broken"), 0644); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	service.rollbackFiles(context.Background(), map[string]*fileRollbackEntry{
		filePath: entry,
	}, []string{filePath})

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(got) != string(originalContent) {
		t.Fatalf("unexpected restored content: got %q want %q", string(got), string(originalContent))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat restored file: %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("unexpected restored mode: got %o want %o", info.Mode().Perm(), os.FileMode(0640))
	}
}

func TestRollbackFilesDeletesNewFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "new.go")

	service := newWorkspaceFileTestService(tempDir)
	entry, err := service.captureFileRollbackEntry(filePath)
	if err != nil {
		t.Fatalf("capture rollback entry: %v", err)
	}
	if entry.Existed {
		t.Fatalf("expected new file rollback entry to mark Existed=false")
	}

	if err := os.WriteFile(filePath, []byte("package newfile\n"), 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	service.rollbackFiles(context.Background(), map[string]*fileRollbackEntry{
		filePath: entry,
	}, []string{filePath})

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected new file to be deleted, got err=%v", err)
	}
}

func TestResolveBatchWriteTargetRejectsInitFile(t *testing.T) {
	t.Parallel()

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.FileWriteItem{
		FullCodePath: "/luobei/demo/ticket_system",
		FileName:     "init_",
		FileType:     "go",
	})
	if err == nil {
		t.Fatalf("expected init_ file to be rejected")
	}
}

func TestResolveBatchWriteTargetRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.FileWriteItem{
		FullCodePath: "/luobei/demo/../code/cmd/app",
		FileName:     "main",
		FileType:     "go",
	})
	if err == nil {
		t.Fatalf("expected path traversal to be rejected")
	}
}

func TestResolveBatchWriteTargetRejectsTraversalInFileType(t *testing.T) {
	t.Parallel()

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.FileWriteItem{
		FullCodePath: "/luobei/demo/ticket_system",
		FileName:     "ticket",
		FileType:     "../go",
	})
	if err == nil {
		t.Fatalf("expected invalid file_type to be rejected")
	}
}

func TestResolveBatchWriteTargetAcceptsBusinessGoFile(t *testing.T) {
	t.Parallel()

	packageDir, filePath, fileName, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.FileWriteItem{
		FullCodePath: "/luobei/demo/ticket_system",
		FileName:     "ticket",
		FileType:     "go",
	})
	if err != nil {
		t.Fatalf("expected valid business file to pass, got err=%v", err)
	}
	if packageDir != "/tmp/api/ticket_system" {
		t.Fatalf("unexpected packageDir: %s", packageDir)
	}
	if filePath != "/tmp/api/ticket_system/ticket.go" {
		t.Fatalf("unexpected filePath: %s", filePath)
	}
	if fileName != "ticket" {
		t.Fatalf("unexpected fileName: %s", fileName)
	}
}

func TestValidateRelativePackagePathRejectsTraversal(t *testing.T) {
	t.Parallel()

	if _, err := validateRelativePackagePath("../cmd/app"); err == nil {
		t.Fatalf("expected traversal package path to be rejected")
	}
}

func TestValidateRelativePackagePathRejectsInvalidGoPackageName(t *testing.T) {
	t.Parallel()

	invalidPaths := []string{"ticket-system", "ticket/type", "ticket/1order"}
	for _, packagePath := range invalidPaths {
		packagePath := packagePath
		t.Run(packagePath, func(t *testing.T) {
			t.Parallel()
			if _, err := validateRelativePackagePath(packagePath); err == nil {
				t.Fatalf("expected invalid package path %q to be rejected", packagePath)
			}
		})
	}
}

func TestBatchCreateDirectoryTreeRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	_, err := service.BatchCreateDirectoryTree(context.Background(), &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryScaffoldItem{
			{
				FullCodePath: "/luobei/demo/../code/cmd/app",
				Name:         "bad",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid directory path to be rejected")
	}
}

func TestBatchCreateDirectoryTreeRejectsInvalidGoPackageName(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	_, err := service.BatchCreateDirectoryTree(context.Background(), &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryScaffoldItem{
			{
				FullCodePath: "/luobei/demo/ticket-system",
				Name:         "bad",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid go package name to be rejected")
	}
}

func TestBatchCreateDirectoryTreeWritesInitFile(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	resp, err := service.BatchCreateDirectoryTree(context.Background(), &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryScaffoldItem{
			{
				FullCodePath: "/luobei/demo/ticket_system/order",
				Name:         "订单",
				Description:  "订单目录",
			},
		},
	})
	if err != nil {
		t.Fatalf("BatchCreateDirectoryTree: %v", err)
	}
	if resp.DirectoryCount != 1 {
		t.Fatalf("unexpected directory count: %d", resp.DirectoryCount)
	}

	initFilePath := filepath.Join(basePath, "luobei", "demo", "code", "api", "ticket_system", "order", "init_.go")
	content, err := os.ReadFile(initFilePath)
	if err != nil {
		t.Fatalf("read init_.go: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "package order") {
		t.Fatalf("unexpected init_.go package declaration: %s", contentStr)
	}
	if !strings.Contains(contentStr, `RouterGroup: "/ticket_system/order"`) {
		t.Fatalf("unexpected init_.go router group: %s", contentStr)
	}
}

func TestDeleteServiceTreeRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	if err := service.DeleteServiceTree(context.Background(), "luobei", "demo", "../cmd/app"); err == nil {
		t.Fatalf("expected invalid delete path to be rejected")
	}
}

func TestBuildBatchWriteFilesRuntimeResp(t *testing.T) {
	t.Parallel()

	service := &WorkspaceChangeService{}
	state := &batchWriteState{
		writtenPaths: []string{"/alice/demo/ticket/list", "/alice/demo/ticket/detail"},
	}
	release := &appReleaseResult{
		oldVersion:    "v3",
		newVersion:    "v4",
		gitCommitHash: "abc123",
	}

	resp := service.buildBatchWriteFilesRuntimeResp(state, release)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.FileCount != 2 {
		t.Fatalf("unexpected file count: %d", resp.FileCount)
	}
	if len(resp.WrittenPaths) != 2 {
		t.Fatalf("unexpected written paths: %#v", resp.WrittenPaths)
	}
	if resp.OldVersion != "v3" || resp.NewVersion != "v4" {
		t.Fatalf("unexpected versions: old=%s new=%s", resp.OldVersion, resp.NewVersion)
	}
	if resp.GitCommitHash != "abc123" {
		t.Fatalf("unexpected commit hash: %s", resp.GitCommitHash)
	}
}

func TestBatchWriteFilesDoesNotMutateWhenAppManageServiceMissing(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	_, err := service.BatchWriteFiles(context.Background(), &dto.BatchWriteFilesRuntimeReq{
		User: "luobei",
		App:  "demo",
		Files: []*dto.FileWriteItem{
			{
				FullCodePath: "/luobei/demo/ticket_system",
				FileName:     "ticket",
				FileType:     "go",
				Content:      "package ticket\n",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected missing appManageService to fail")
	}

	filePath := filepath.Join(basePath, "luobei", "demo", "code", "api", "ticket_system", "ticket.go")
	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected file to remain absent, got err=%v", statErr)
	}
}

func TestBatchWriteFilesRejectsMissingAppWithoutCreatingDirectories(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestServiceWithAppManage(basePath)

	_, err := service.BatchWriteFiles(context.Background(), &dto.BatchWriteFilesRuntimeReq{
		User: "luobei",
		App:  "demo",
		Files: []*dto.FileWriteItem{
			{
				FullCodePath: "/luobei/demo/ticket_system",
				FileName:     "ticket",
				FileType:     "go",
				Content:      "package ticket\n",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected missing app to fail")
	}

	appDir := newRuntimeAppPaths(basePath, "luobei", "demo").AppDir()
	if _, statErr := os.Stat(appDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected app dir to remain absent, got err=%v", statErr)
	}
}

func TestWriteSourceFilesRejectsMissingAppWithoutMutatingDisk(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	workspaceFiles := newWorkspaceFileTestService(basePath)

	_, err := workspaceFiles.writeSourceFiles(context.Background(), "luobei", "demo", []*dto.SourceFileWrite{
		{
			DirectoryPath: "ticket_system",
			FileName:      "ticket",
			SourceCode:    "package ticket\n",
		},
	})
	if err == nil {
		t.Fatalf("expected missing app to fail")
	}

	appDir := newRuntimeAppPaths(basePath, "luobei", "demo").AppDir()
	if _, statErr := os.Stat(appDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected app dir to remain absent, got err=%v", statErr)
	}
}

func TestWriteSourceFilesRejectsSQLiteDriverImportWithoutMutatingDisk(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	workspaceFiles := newWorkspaceFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "luobei", "demo")
	if err := os.MkdirAll(appPaths.AppDir(), 0755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	_, err := workspaceFiles.writeSourceFiles(context.Background(), "luobei", "demo", []*dto.SourceFileWrite{
		{
			DirectoryPath: "ticket_system",
			FileName:      "importer",
			SourceCode: `package ticket_system

import _ "github.com/mattn/go-sqlite3"
`,
		},
	})
	if err == nil {
		t.Fatal("expected sqlite driver import to fail")
	}
	if !strings.Contains(err.Error(), "源码规范校验失败") || !strings.Contains(err.Error(), "KageOS SDK 已全局注册") {
		t.Fatalf("unexpected error: %v", err)
	}

	filePath := filepath.Join(appPaths.APIDir(), "ticket_system", "importer.go")
	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected rejected file to remain absent, got err=%v", statErr)
	}
	packageDir := filepath.Dir(filePath)
	if _, statErr := os.Stat(packageDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected rejected package dir to remain absent, got err=%v", statErr)
	}
}

func TestWriteBatchFilesToDiskRollsBackOnError(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	workspaceFiles := newWorkspaceFileTestService(basePath)
	apiDir := newRuntimeAppPaths(basePath, "luobei", "demo").APIDir()
	filePath := filepath.Join(apiDir, "ticket_system", "ticket.go")
	appDir := newRuntimeAppPaths(basePath, "luobei", "demo").AppDir()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	originalContent := []byte("package ticket\n\nconst Name = \"old\"\n")
	if err := os.WriteFile(filePath, originalContent, 0644); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	_, err := workspaceFiles.writeDirectoryTreeFiles(context.Background(), "luobei", "demo", []*dto.FileWriteItem{
		{
			FullCodePath: "/luobei/demo/ticket_system",
			FileName:     "ticket",
			FileType:     "go",
			Content:      "package ticket\n\nconst Name = \"new\"\n",
		},
		{
			FullCodePath: "/luobei/demo/../cmd/app",
			FileName:     "main",
			FileType:     "go",
			Content:      "package main\n",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid path to fail")
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read rolled back file: %v", err)
	}
	if string(got) != string(originalContent) {
		t.Fatalf("unexpected rolled back content: got %q want %q", string(got), string(originalContent))
	}
}

func TestWriteDirectoryTreeFilesRollsBackCreatedDirectoriesOnError(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	workspaceFiles := newWorkspaceFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "luobei", "demo")

	if err := os.MkdirAll(appPaths.AppDir(), 0755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	_, err := workspaceFiles.writeDirectoryTreeFiles(context.Background(), "luobei", "demo", []*dto.FileWriteItem{
		{
			FullCodePath: "/luobei/demo/ticket_system",
			FileName:     "ticket",
			FileType:     "go",
			Content:      "package ticket\n",
		},
		{
			FullCodePath: "/luobei/demo/../cmd/app",
			FileName:     "main",
			FileType:     "go",
			Content:      "package main\n",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid path to fail")
	}

	packageDir := filepath.Join(appPaths.APIDir(), "ticket_system")
	if _, statErr := os.Stat(packageDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected package dir to be removed, got err=%v", statErr)
	}
	if _, statErr := os.Stat(appPaths.APIDir()); !os.IsNotExist(statErr) {
		t.Fatalf("expected api dir to be removed, got err=%v", statErr)
	}
}

func TestDirectoryReplaceRollbackRestoresOldDirectoryAndMain(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	workspaceFiles := newWorkspaceFileTestService(basePath)
	appPaths := newRuntimeAppPaths(basePath, "luobei", "demo")
	targetDir := filepath.Join(appPaths.APIDir(), "tools", "a")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	oldFile := filepath.Join(targetDir, "old.go")
	if err := os.WriteFile(oldFile, []byte("package a\n\nconst Old = true\n"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	mainPath := appPaths.MainGoPath()
	originalMain := []byte("package main\n\nimport _ \"github.com/kageos/kageos/namespace/luobei/demo/code/api/tools/a\"\n")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.WriteFile(mainPath, originalMain, 0644); err != nil {
		t.Fatalf("write main: %v", err)
	}

	state, _, err := workspaceFiles.beginDirectoryReplace(context.Background(), "luobei", "demo", "/luobei/demo/tools/a")
	if err != nil {
		t.Fatalf("beginDirectoryReplace: %v", err)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("expected old target to be moved out of compile path, got err=%v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir replacement dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "new.go"), []byte("package a\n\nconst New = true\n"), 0644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	workspaceFiles.rollbackDirectoryReplace(context.Background(), state)

	gotOld, err := os.ReadFile(oldFile)
	if err != nil {
		t.Fatalf("read restored old file: %v", err)
	}
	if !strings.Contains(string(gotOld), "Old = true") {
		t.Fatalf("unexpected restored old file: %s", string(gotOld))
	}
	if _, err := os.Stat(filepath.Join(targetDir, "new.go")); !os.IsNotExist(err) {
		t.Fatalf("expected replacement file to be removed, got err=%v", err)
	}
	gotMain, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read restored main: %v", err)
	}
	if string(gotMain) != string(originalMain) {
		t.Fatalf("main.go was not restored: got %q want %q", string(gotMain), string(originalMain))
	}
}
