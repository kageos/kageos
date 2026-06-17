package app

import (
	"database/sql"
	"slices"
	"strings"
	"testing"
)

func TestSQLite3DatabaseSQLDriverIsRegisteredByDefault(t *testing.T) {
	if !slices.Contains(sql.Drivers(), "sqlite3") {
		t.Fatal("sqlite3 driver should be registered by the SDK for uploaded SQLite files")
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite3: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite3: %v", err)
	}
}

func TestGetDBByPackagePathRequiresContextCapability(t *testing.T) {
	db, err := GetDBByPackagePath("sales/leads")
	if err == nil {
		t.Fatal("GetDBByPackagePath should reject capability-less business DB access")
	}
	if db != nil {
		t.Fatalf("expected nil db, got %#v", db)
	}
	if !strings.Contains(err.Error(), "ctx.GetGormDB()") {
		t.Fatalf("unexpected error: %v", err)
	}
}
