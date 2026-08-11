package jsonx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFileDoesNotTruncateExistingFileWhenMarshalFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "value.json")
	if err := SaveFile(path, map[string]string{"status": "kept"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(path, make(chan int)); err == nil {
		t.Fatal("SaveFile must reject unsupported JSON values")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"status":"kept"}`; got != want {
		t.Fatalf("file content = %q, want %q", got, want)
	}
}
