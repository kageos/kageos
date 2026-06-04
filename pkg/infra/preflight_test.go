package infra

import (
	"os"
	"path/filepath"
	"testing"
)

func setKageosMode(t *testing.T, mode string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("KAGEOS_ROOT", root)
	if mode == "" {
		return
	}
	stateDir := filepath.Join(root, ".kageos")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "kageos.env"), []byte("KAGEOS_MODE="+mode+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLPreflightAddressDevDefaultsTo3318(t *testing.T) {
	setKageosMode(t, "dev")
	t.Setenv("MYSQL_HOST", "")
	t.Setenv("MYSQL_PORT", "")

	if got, want := mysqlPreflightAddress(), "127.0.0.1:3318"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMySQLPreflightAddressAllowsEnvOverride(t *testing.T) {
	setKageosMode(t, "dev")
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "4406")

	if got, want := mysqlPreflightAddress(), "localhost:4406"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMySQLPreflightAddressProdDefaultsTo3306(t *testing.T) {
	setKageosMode(t, "prod")
	t.Setenv("MYSQL_HOST", "")
	t.Setenv("MYSQL_PORT", "")

	if got, want := mysqlPreflightAddress(), "127.0.0.1:3306"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}
