package infra

import "testing"

func TestMySQLPreflightAddressDevDefaultsTo3318(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("MYSQL_HOST", "")
	t.Setenv("MYSQL_PORT", "")

	if got, want := mysqlPreflightAddress(), "127.0.0.1:3318"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMySQLPreflightAddressAllowsEnvOverride(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "4406")

	if got, want := mysqlPreflightAddress(), "localhost:4406"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMySQLPreflightAddressProdDefaultsTo3306(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("MYSQL_HOST", "")
	t.Setenv("MYSQL_PORT", "")

	if got, want := mysqlPreflightAddress(), "127.0.0.1:3306"; got != want {
		t.Fatalf("mysqlPreflightAddress() = %q, want %q", got, want)
	}
}
