package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kageos/kageos/pkg/sdkmodule"
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

func TestGoModHasSDKReplace(t *testing.T) {
	t.Parallel()

	goMod := []byte(`module demo

go 1.25.0

replace github.com/kageos/kageos-sdk => ../../../../kageos-sdk

require github.com/kageos/kageos-sdk v0.0.0
`)

	if !goModHasSDKReplace("go.mod", goMod) {
		t.Fatalf("goModHasSDKReplace() = false, want true for %s replace", sdkmodule.ModulePath)
	}
}

func TestGoModHasSDKReplaceFalseWithoutReplace(t *testing.T) {
	t.Parallel()

	goMod := []byte(`module demo

go 1.25.0

require github.com/kageos/kageos-sdk v0.2.1
`)

	if goModHasSDKReplace("go.mod", goMod) {
		t.Fatal("goModHasSDKReplace() = true, want false without replace")
	}
}
