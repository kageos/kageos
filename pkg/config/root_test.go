package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverKageOSRootFromUpwardMarker(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	root := filepath.Join(base, "workspace", "repo")
	start := filepath.Join(root, "pkg", "config")

	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir start: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, MarkerKageOSRoot), []byte("marker"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	got := discoverKageOSRootFrom(start)
	if got != root {
		t.Fatalf("discoverKageOSRootFrom(%q) = %q, want %q", start, got, root)
	}
}

func TestDiscoverKageOSRootFromDownwardMarker(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	searchStart := filepath.Join(base, "Library", "Caches", "JetBrains", "tmp")
	root := filepath.Join(base, "Documents", "work", "code", "gitee.com", "kageos")

	if err := os.MkdirAll(searchStart, 0o755); err != nil {
		t.Fatalf("mkdir searchStart: %v", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, MarkerKageOSRoot), []byte("marker"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	got := discoverKageOSRootFrom(searchStart)
	if got != root {
		t.Fatalf("discoverKageOSRootFrom(%q) = %q, want %q", searchStart, got, root)
	}
}
