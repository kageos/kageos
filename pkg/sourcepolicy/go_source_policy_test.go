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
			for _, want := range []string{"kageos SDK 已全局注册", "sql.Open(\"sqlite3\", path)", "sql: Register called twice"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in %v", want, err)
				}
			}
		})
	}
}

func TestValidateAppGoSourceRejectsMainRepoImports(t *testing.T) {
	tests := []string{
		`"github.com/kageos/kageos/sdk/agent-app/app"`,
		`"github.com/kageos/kageos/pkg/logger"`,
		`"github.com/kageos/kageos/pkg/gormx/query"`,
		`"github.com/kageos/kageos/dto"`,
		`"github.com/kageos/kageos/core/app-server/model"`,
	}

	for _, importLine := range tests {
		t.Run(importLine, func(t *testing.T) {
			source := "package demo\n\nimport " + importLine + "\n"
			err := ValidateAppGoSource("bad_import.go", source)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "应用代码只能依赖 github.com/kageos/kageos-sdk") {
				t.Fatalf("expected SDK boundary hint in %v", err)
			}
		})
	}
}

func TestValidateAppGoSourceAllowsKageosSDKImports(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
)

func handler(ctx *app.Context) error {
	_ = query.PageSortReq{}
	logger.Infof(ctx.Context, "ok")
	return nil
}
`
	if err := ValidateAppGoSource("good_import.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
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

	"github.com/kageos/kageos-sdk/agent-app/app"
	"gorm.io/gorm"
)

type Ticket struct {
	ID        int64
	Title     string
	DeletedAt gorm.DeletedAt
	DeletedBy string
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
	return markDeleted(db, 1, ctx.GetRequestUser())
}

func markDeleted(db *gorm.DB, id int64, user string) error {
	return db.Model(&Ticket{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted_at": time.Now(),
		"deleted_by": user,
	}).Error
}
`
	if err := ValidateAppGoSource("good.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceAllowsAppDBPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	db := ctx.GetGormDB()
	return third.Use(db)
}
`
	if err := ValidateAppGoSource("db_external.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceAllowsDirectGetGormDBPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	return third.Use(ctx.GetGormDB())
}
`
	if err := ValidateAppGoSource("db_direct.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceAllowsAppDBWrappedAsAnyPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func handler(ctx *app.Context) error {
	db := ctx.GetGormDB()
	return third.Use(any(db))
}
`
	if err := ValidateAppGoSource("db_any.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceAllowsAppDBMethods(t *testing.T) {
	tests := map[string]string{
		"Exec":        `return ctx.GetGormDB().Exec("DELETE FROM tickets").Error`,
		"Unscoped":    `return ctx.GetGormDB().Unscoped().Where("id = ?", 1).Delete(&Ticket{}).Error`,
		"Migrator":    `return ctx.GetGormDB().Migrator().DropTable(&Ticket{})`,
		"DB":          `_, err := ctx.GetGormDB().DB(); return err`,
		"AutoMigrate": `return ctx.GetGormDB().AutoMigrate(&Ticket{})`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			source := `package demo

import "github.com/kageos/kageos-sdk/agent-app/app"

type Ticket struct{}

func handler(ctx *app.Context) error {
	` + body + `
}
`
			if err := ValidateAppGoSource("db_methods.go", source); err != nil {
				t.Fatalf("ValidateAppGoSource() error = %v", err)
			}
		})
	}
}

func TestValidateAppGoSourceAllowsReadOnlyRawAppDBQuery(t *testing.T) {
	source := `package demo

import "github.com/kageos/kageos-sdk/agent-app/app"

const ticketStatsSQL = ` + "`" + `
WITH stats AS (
	SELECT status, COUNT(*) AS count
	FROM tickets
	WHERE deleted_at IS NULL AND created_by = ?
	GROUP BY status
)
SELECT status, count FROM stats WHERE count > ?
` + "`" + `

type TicketStat struct {
	Status string
	Count int64
}

func handler(ctx *app.Context, owner string) error {
	var rows []TicketStat
	return ctx.GetGormDB().Raw(ticketStatsSQL, owner, 0).Scan(&rows).Error
}
`
	if err := ValidateAppGoSource("good.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}

func TestValidateAppGoSourceAllowsRawAppDBQueryForms(t *testing.T) {
	tests := map[string]string{
		"update":  `return ctx.GetGormDB().Raw("UPDATE tickets SET status = ? WHERE id = ?", "done", 1).Error`,
		"delete":  `return ctx.GetGormDB().Raw("DELETE FROM tickets WHERE id = ?", 1).Error`,
		"ddl":     `return ctx.GetGormDB().Raw("CREATE TEMPORARY TABLE tmp_stats (id bigint)").Error`,
		"set":     `return ctx.GetGormDB().Raw("SET sql_mode = ''").Error`,
		"dynamic": `sql := "SELECT * FROM tickets"; return ctx.GetGormDB().Raw(sql).Error`,
		"concat":  `return ctx.GetGormDB().Raw("SELECT * FROM tickets ORDER BY " + orderBy).Error`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			source := `package demo

import "github.com/kageos/kageos-sdk/agent-app/app"

func handler(ctx *app.Context, orderBy string) error {
	` + body + `
}
`
			if err := ValidateAppGoSource("raw.go", source); err != nil {
				t.Fatalf("ValidateAppGoSource() error = %v", err)
			}
		})
	}
}

func TestValidateAppGoSourceAllowsAppDBStoredOrReturned(t *testing.T) {
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
	"github.com/kageos/kageos-sdk/agent-app/app"
	"gorm.io/gorm"
)

type Holder struct{ DB *gorm.DB }
var GlobalDB *gorm.DB

func handler(ctx *app.Context, holder *Holder, m map[string]any) *gorm.DB {
	` + body + `
	return nil
}
`
			if err := ValidateAppGoSource("db_storage.go", source); err != nil {
				t.Fatalf("ValidateAppGoSource() error = %v", err)
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
