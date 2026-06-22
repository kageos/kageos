package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateVersionUsesProvidedReleasesDir(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	legacyReleasesDir := filepath.Join(workDir, "namespace", "alice", "demo", "workplace", "bin", "releases")
	if err := os.MkdirAll(legacyReleasesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyReleasesDir, "alice_demo_v9"), []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}

	releasesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(releasesDir, "alice_demo_v2"), []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}

	got := NewBuilder(workDir).generateVersion("alice", "demo", releasesDir)
	if got != "v3" {
		t.Fatalf("generateVersion() = %q, want v3", got)
	}
}
