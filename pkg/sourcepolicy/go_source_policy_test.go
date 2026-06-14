package sourcepolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAppGoSourceAllowsDatabaseSQLOpenSQLite3(t *testing.T) {
	source := `package demo

import "database/sql"

func openUploadedSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}
`
	if err := ValidateAppGoSource("importer.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceRejectsSQLiteDriverImports(t *testing.T) {
	tests := []string{
		`_ "github.com/mattn/go-sqlite3"`,
		`_ "github.com/ncruces/go-sqlite3/driver"`,
		`"gorm.io/driver/sqlite"`,
	}

	for _, importLine := range tests {
		t.Run(importLine, func(t *testing.T) {
			source := "package demo\n\nimport " + importLine + "\n"
			err := ValidateAppGoSource("bad.go", source)
			if err == nil {
				t.Fatal("expected validation error")
			}
			for _, want := range []string{"KageOS SDK 已全局注册", "sql.Open(\"sqlite3\", path)", "sql: Register called twice"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in %v", want, err)
				}
			}
		})
	}
}

func TestValidateAppGoSourceRejectsSQLRegister(t *testing.T) {
	source := `package demo

import "database/sql"

func init() {
	sql.Register("sqlite3", nil)
}
`
	err := ValidateAppGoSource("bad.go", source)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "禁止调用 database/sql.Register") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAppGoSourceDirScansCodeRootFromCmdApp(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "code", "cmd", "app")
	apiDir := filepath.Join(root, "code", "api", "demo")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("mkdir api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "bad.go"), []byte("package demo\n\nimport _ \"github.com/mattn/go-sqlite3\"\n"), 0644); err != nil {
		t.Fatalf("write bad: %v", err)
	}

	err := ValidateAppGoSourceDir(sourceDir)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), filepath.Join("api", "demo", "bad.go")) {
		t.Fatalf("expected bad file path in error, got %v", err)
	}
}
