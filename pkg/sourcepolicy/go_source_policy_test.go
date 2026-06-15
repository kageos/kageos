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

func TestValidateAppGoSourceAllowsLocalGORMUsage(t *testing.T) {
	source := `package demo

import (
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"gorm.io/gorm"
)

type Ticket struct {
	ID        int64
	Title     string
	DeletedAt gorm.DeletedAt
}

func listTickets(ctx *app.Context) error {
	db := ctx.GetGormDB()
	query := db.Model(&Ticket{}).Where("title LIKE ?", "%hello%")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}
	var rows []Ticket
	if err := query.Order("id desc").Find(&rows).Error; err != nil {
		return err
	}
	return markDeleted(db, 1)
}

func markDeleted(db *gorm.DB, id int64) error {
	return db.Model(&Ticket{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}
`
	if err := ValidateAppGoSource("good.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceRejectsAppDBPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	db := ctx.GetGormDB()
	return third.Use(db)
}
`
	err := ValidateAppGoSource("bad.go", source)
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{"禁止把应用数据库对象传给第三方库", "ctx.GetGormDB() 得到的数据库对象只能在当前目录业务代码内直接使用"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %v", want, err)
		}
	}
}

func TestValidateAppGoSourceRejectsDirectGetGormDBPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	return third.Use(ctx.GetGormDB())
}
`
	err := ValidateAppGoSource("bad.go", source)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "禁止把应用数据库对象传给第三方库") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAppGoSourceRejectsAppDBWrappedAsAnyPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	db := ctx.GetGormDB()
	return third.Use(any(db))
}
`
	err := ValidateAppGoSource("bad.go", source)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "禁止把应用数据库对象传给第三方库") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAppGoSourceRejectsDangerousAppDBMethods(t *testing.T) {
	tests := map[string]string{
		"Exec":        `return ctx.GetGormDB().Exec("DELETE FROM tickets").Error`,
		"Raw":         `return ctx.GetGormDB().Raw("SELECT * FROM tickets").Error`,
		"Unscoped":    `return ctx.GetGormDB().Unscoped().Where("id = ?", 1).Delete(&Ticket{}).Error`,
		"Migrator":    `return ctx.GetGormDB().Migrator().DropTable(&Ticket{})`,
		"DB":          `_, err := ctx.GetGormDB().DB(); return err`,
		"AutoMigrate": `return ctx.GetGormDB().AutoMigrate(&Ticket{})`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			source := `package demo

import "github.com/kageos/kageos/sdk/agent-app/app"

type Ticket struct{}

func handler(ctx *app.Context) error {
	` + body + `
}
`
			err := ValidateAppGoSource("bad.go", source)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "禁止使用 db."+name) && !strings.Contains(err.Error(), "禁止在业务代码中调用 db."+name) {
				t.Fatalf("expected method %s in error, got %v", name, err)
			}
		})
	}
}

func TestValidateAppGoSourceRejectsAppDBStoredOrReturned(t *testing.T) {
	tests := map[string]string{
		"struct_field":     `holder.DB = ctx.GetGormDB()`,
		"map_value":        `m["db"] = ctx.GetGormDB()`,
		"composite_struct": `_ = Holder{DB: ctx.GetGormDB()}`,
		"composite_slice":  `_ = []any{ctx.GetGormDB()}`,
		"global_var":       `GlobalDB = ctx.GetGormDB()`,
		"return_db":        `return ctx.GetGormDB()`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			source := `package demo

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
	"gorm.io/gorm"
)

type Holder struct{ DB *gorm.DB }
var GlobalDB *gorm.DB

func handler(ctx *app.Context, holder *Holder, m map[string]any) *gorm.DB {
	` + body + `
	return nil
}
`
			err := ValidateAppGoSource("bad.go", source)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "应用数据库对象") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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
