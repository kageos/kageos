package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicReplacesContentAndPermission(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "metadata", "version.json")
	if err := writeFileAtomic(filePath, []byte("old"), 0644); err != nil {
		t.Fatalf("initial writeFileAtomic: %v", err)
	}

	if err := writeFileAtomic(filePath, []byte("new"), 0600); err != nil {
		t.Fatalf("replace writeFileAtomic: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("unexpected file content: %q", string(data))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("unexpected permission: %o", info.Mode().Perm())
	}
}
