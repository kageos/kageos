package app

import (
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
)

func TestSanitizeDBName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "adds suffix", in: "orders", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "keeps db suffix", in: "orders.db", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "drops traversal directory", in: "../secrets/orders.db", want: filepath.Join(getDataDir(), "orders.db")},
		{name: "uses base name", in: "nested/orders.db", want: filepath.Join(getDataDir(), "orders.db")},
	}

	for _, tt := range tests {
		if got := sanitizeDBName(tt.in); got != tt.want {
			t.Fatalf("%s: want %s, got %s", tt.name, tt.want, got)
		}
	}
}

func TestDBLogFilePath(t *testing.T) {
	got := dbLogFilePath(filepath.Join("/tmp", "agent", "orders.db"))
	want := filepath.Join("/tmp", "agent", "orders.log")
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

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
