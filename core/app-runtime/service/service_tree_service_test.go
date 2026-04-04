package service

import (
	"context"
	"os"
	"path/filepath"
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
