package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSystemDirectorySeedTargetPathUsesDirectoryAsTargetNode(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "system-seed")
	filePath := filepath.Join(seedDir, "system", "tools", "openapi", "excel.json")

	got, err := systemDirectorySeedTargetPath(seedDir, filePath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/system/tools/openapi" {
		t.Fatalf("target path = %q, want /system/tools/openapi", got)
	}
}

func TestSystemDirectorySeedTargetPathRejectsRootJSON(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "system-seed")
	_, err := systemDirectorySeedTargetPath(seedDir, filepath.Join(seedDir, "excel.json"))
	if err == nil {
		t.Fatal("expected root json to be rejected")
	}
}

func TestListSystemDirectorySeedFilesSorted(t *testing.T) {
	seedDir := t.TempDir()
	for _, rel := range []string{"system/tools/z.json", "system/tools/a.json", "system/tools/readme.md", "system/openapi/platform.json"} {
		path := filepath.Join(seedDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := listSystemDirectorySeedFiles(seedDir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(seedDir, file)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, filepath.ToSlash(rel))
	}
	want := []string{"system/openapi/platform.json", "system/tools/a.json", "system/tools/z.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestSystemSeedBundlesUseCapabilitySchema(t *testing.T) {
	seedDir := filepath.Join("..", "system-seed")
	files, err := listSystemDirectorySeedFiles(seedDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected system seed bundles")
	}

	for _, file := range files {
		if _, err := systemDirectorySeedTargetPath(seedDir, file); err != nil {
			t.Fatalf("invalid seed target for %s: %v", file, err)
		}
		bundle, err := readCapabilityBundleFile(file)
		if err != nil {
			t.Fatalf("read seed bundle %s: %v", file, err)
		}
		if err := validateCapabilityBundle(bundle); err != nil {
			t.Fatalf("invalid capability seed %s: %v", file, err)
		}
	}
}
