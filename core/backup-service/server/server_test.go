package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
)

func TestBackupServerBasicAuthProtectsConsoleAndAPI(t *testing.T) {
	t.Parallel()

	cfg := newTestServerConfig(t)
	cfg.Auth.Username = "admin"
	cfg.Auth.Password = "admin123"
	cfg.Auth.Realm = "Backup Control Plane"

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer func() {
		_ = srv.controlPlane.Close()
	}()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/backup", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /backup status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/backup/api/v1/status", nil)
	rec = httptest.NewRecorder()
	srv.httpServer.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /backup/api/v1/status status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/backup", nil)
	req.SetBasicAuth("admin", "admin123")
	rec = httptest.NewRecorder()
	srv.httpServer.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /backup with auth status = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/backup/api/v1/status", nil)
	req.SetBasicAuth("admin", "admin123")
	rec = httptest.NewRecorder()
	srv.httpServer.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /backup/api/v1/status with auth status = %d, want 200", rec.Code)
	}
}

func newTestServerConfig(t *testing.T) *config.BackupServiceConfig {
	t.Helper()

	tmpDir := t.TempDir()
	cfg := &config.BackupServiceConfig{}
	cfg.Storage.Root = tmpDir
	cfg.Storage.NamespacePath = filepath.Join(tmpDir, "namespace")
	cfg.Storage.DataPath = filepath.Join(tmpDir, "data")
	cfg.Storage.LogsPath = filepath.Join(tmpDir, "logs")
	cfg.Storage.MySQLPath = filepath.Join(tmpDir, "mysql")
	cfg.Storage.MinIOPath = filepath.Join(tmpDir, "minio")
	cfg.Storage.PodmanStoragePath = filepath.Join(tmpDir, "podman")
	cfg.Repository.RootPath = filepath.Join(tmpDir, "repo")
	cfg.Repository.StatePath = filepath.Join(tmpDir, "state")
	cfg.Repository.StagingPath = filepath.Join(tmpDir, "staging")
	cfg.Database.Path = filepath.Join(tmpDir, "state", "backup-service.db")
	return cfg
}
