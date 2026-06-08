package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWorkspaceModeFromRootUsesEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".kageos"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kageos", "kageos.env"), []byte("KAGEOS_MODE=dev\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if got := readWorkspaceModeFromRoot(root); got != ConfigModeDev {
		t.Fatalf("readWorkspaceModeFromRoot() = %q, want dev", got)
	}
}

func TestReadWorkspaceModeFromRootRejectsUnknownMode(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".kageos"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kageos", "kageos.env"), []byte("KAGEOS_MODE=staging\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if got := readWorkspaceModeFromRoot(root); got != "" {
		t.Fatalf("readWorkspaceModeFromRoot() = %q, want empty", got)
	}
}

func TestGetDevEngineUsesWorkspaceEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KAGEOS_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, ".kageos"), 0755); err != nil {
		t.Fatal(err)
	}
	env := "KAGEOS_MODE=dev\nKAGEOS_DEV_ENGINE=docker\n"
	if err := os.WriteFile(filepath.Join(root, ".kageos", "kageos.env"), []byte(env), 0600); err != nil {
		t.Fatal(err)
	}

	if got := GetDevEngine(); got != ConfigDevEngineDocker {
		t.Fatalf("GetDevEngine() = %q, want docker", got)
	}
}

func TestGetDevEngineDefaultsToPodmanInDev(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KAGEOS_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, ".kageos"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kageos", "kageos.env"), []byte("KAGEOS_MODE=dev\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if got := GetDevEngine(); got != ConfigDevEnginePodman {
		t.Fatalf("GetDevEngine() = %q, want podman", got)
	}
}
