package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestRollbackFilesRestoresOverwrittenFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ticket.go")
	originalContent := []byte("package ticket\n\nconst Name = \"old\"\n")

	if err := os.WriteFile(filePath, originalContent, 0640); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	service := &ServiceTreeService{}
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

	service := &ServiceTreeService{}
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

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.DirectoryTreeItem{
		Type:         "file",
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

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.DirectoryTreeItem{
		Type:         "file",
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

	_, _, _, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.DirectoryTreeItem{
		Type:         "file",
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

	packageDir, filePath, fileName, err := resolveBatchWriteTarget("luobei", "demo", "/tmp/api", &dto.DirectoryTreeItem{
		Type:         "file",
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

func TestBatchCreateDirectoryTreeRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	_, err := service.BatchCreateDirectoryTree(context.Background(), &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryTreeItem{
			{
				Type:         "directory",
				FullCodePath: "/luobei/demo/../code/cmd/app",
				Name:         "bad",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid directory path to be rejected")
	}
}

func TestBatchCreateDirectoryTreeWritesInitFile(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	resp, err := service.BatchCreateDirectoryTree(context.Background(), &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryTreeItem{
			{
				Type:         "directory",
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

func TestBatchWriteFilesDoesNotMutateWhenAppManageServiceMissing(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)

	_, err := service.BatchWriteFiles(context.Background(), &dto.BatchWriteFilesRuntimeReq{
		User: "luobei",
		App:  "demo",
		Files: []*dto.DirectoryTreeItem{
			{
				Type:         "file",
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

func TestWriteBatchFilesToDiskRollsBackOnError(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	service := newServiceTreeTestService(basePath)
	apiDir := newRuntimeAppPaths(basePath, "luobei", "demo").APIDir()
	filePath := filepath.Join(apiDir, "ticket_system", "ticket.go")

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}

	originalContent := []byte("package ticket\n\nconst Name = \"old\"\n")
	if err := os.WriteFile(filePath, originalContent, 0644); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	_, err := service.writeBatchFilesToDisk(context.Background(), "luobei", "demo", apiDir, []*dto.DirectoryTreeItem{
		{
			Type:         "file",
			FullCodePath: "/luobei/demo/ticket_system",
			FileName:     "ticket",
			FileType:     "go",
			Content:      "package ticket\n\nconst Name = \"new\"\n",
		},
		{
			Type:         "file",
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
