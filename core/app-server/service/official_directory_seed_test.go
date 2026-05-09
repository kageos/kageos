package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOfficialDirectorySeedTargetPathUsesFirstDirectoryAsSystemApp(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "official-seed")
	filePath := filepath.Join(seedDir, "tools", "excel.json")

	got, err := officialDirectorySeedTargetPath(seedDir, filePath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/system/tools" {
		t.Fatalf("target path = %q, want /system/tools", got)
	}
}

func TestOfficialDirectorySeedTargetPathRejectsRootJSON(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "official-seed")
	_, err := officialDirectorySeedTargetPath(seedDir, filepath.Join(seedDir, "excel.json"))
	if err == nil {
		t.Fatal("expected root json to be rejected")
	}
}

func TestListOfficialDirectorySeedFilesSorted(t *testing.T) {
	seedDir := t.TempDir()
	for _, rel := range []string{"tools/z.json", "tools/a.json", "tools/readme.md", "openapi/platform.json"} {
		path := filepath.Join(seedDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := listOfficialDirectorySeedFiles(seedDir)
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
	want := []string{"openapi/platform.json", "tools/a.json", "tools/z.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}
