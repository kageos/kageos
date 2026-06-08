package infra

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestMinIOPreflightAddressDefaultsTo9000(t *testing.T) {
	t.Setenv("MINIO_HOST", "")
	t.Setenv("MINIO_PORT", "")

	if got, want := minioPreflightAddress(), "127.0.0.1:9000"; got != want {
		t.Fatalf("minioPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMinIOPreflightAddressAllowsEnvOverride(t *testing.T) {
	t.Setenv("MINIO_HOST", "minio.internal")
	t.Setenv("MINIO_PORT", "9443")

	if got, want := minioPreflightAddress(), "minio.internal:9443"; got != want {
		t.Fatalf("minioPreflightAddress() = %q, want %q", got, want)
	}
}

func TestMinIOHealthURL(t *testing.T) {
	if got, want := MinIOHealthURL("127.0.0.1:9000", false), "http://127.0.0.1:9000/minio/health/ready"; got != want {
		t.Fatalf("MinIOHealthURL(http) = %q, want %q", got, want)
	}
	if got, want := MinIOHealthURL("minio.example.com", true), "https://minio.example.com/minio/health/ready"; got != want {
		t.Fatalf("MinIOHealthURL(https) = %q, want %q", got, want)
	}
}

func TestCheckMinIOClockSkewPassesWithinThreshold(t *testing.T) {
	serverTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", serverTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	err := CheckMinIOClockSkew(context.Background(), ts.URL, 15*time.Minute, func() time.Time {
		return serverTime.Add(5 * time.Minute)
	})
	if err != nil {
		t.Fatalf("CheckMinIOClockSkew() unexpected error: %v", err)
	}
}

func TestCheckMinIOClockSkewFailsOverThreshold(t *testing.T) {
	serverTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", serverTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	err := CheckMinIOClockSkew(context.Background(), ts.URL, 15*time.Minute, func() time.Time {
		return serverTime.Add(16 * time.Minute)
	})
	if err == nil {
		t.Fatal("expected clock skew error")
	}
	var skewErr *MinIOClockSkewError
	if !errors.As(err, &skewErr) {
		t.Fatalf("expected MinIOClockSkewError, got %T: %v", err, err)
	}
	if skewErr.Skew != 16*time.Minute {
		t.Fatalf("unexpected skew: %s", skewErr.Skew)
	}
}

func TestCheckMinIOClockSkewFallsBackToRootWhenHealthHasNoDate(t *testing.T) {
	serverTime := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/minio/health/live" {
			w.Header()["Date"] = nil
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Date", serverTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	err := CheckMinIOClockSkew(context.Background(), ts.URL+"/minio/health/live", 15*time.Minute, func() time.Time {
		return serverTime.Add(5 * time.Minute)
	})
	if err != nil {
		t.Fatalf("CheckMinIOClockSkew() unexpected error: %v", err)
	}
}
