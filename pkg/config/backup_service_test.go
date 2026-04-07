package config

import (
	"path/filepath"
	"testing"
)

func TestBackupServiceConfigNormalizeForHostStorageRoot(t *testing.T) {
	t.Parallel()

	storageRoot := filepath.Join(t.TempDir(), "storage-root")

	cfg := &BackupServiceConfig{}
	cfg.Storage.Root = "${STORAGE_ROOT}"
	cfg.Storage.NamespacePath = "/app/namespace"
	cfg.Storage.DataPath = "/app/data"
	cfg.Storage.LogsPath = "/app/logs"
	cfg.Storage.MySQLPath = "/storage/mysql"
	cfg.Storage.MinIOPath = "/storage/minio"
	cfg.Storage.PodmanStoragePath = "/storage/podman_storage"
	cfg.Repository.RootPath = "/app/data/backup/repo"
	cfg.Repository.StatePath = "/app/data/backup/state"
	cfg.Repository.StagingPath = "/app/data/backup/staging"
	cfg.Database.Path = "/app/data/backup/state/backup-service.db"
	cfg.Maintenance.MarkerPath = "/app/data/backup/state/maintenance.flag"
	cfg.Maintenance.PagePath = "/app/data/backup/state/maintenance.html"
	cfg.Maintenance.MetadataPath = "/app/data/backup/state/maintenance.json"
	cfg.Dependencies.MySQLAddress = "mysql:3306"
	cfg.Dependencies.MinIOAddress = "minio:9000"

	cfg.normalizeForHostStorageRoot(storageRoot)

	if got := cfg.Storage.Root; got != storageRoot {
		t.Fatalf("Storage.Root = %q, want %q", got, storageRoot)
	}
	if got := cfg.Storage.NamespacePath; got != filepath.Join(storageRoot, "namespace") {
		t.Fatalf("NamespacePath = %q", got)
	}
	if got := cfg.Storage.DataPath; got != filepath.Join(storageRoot, "data") {
		t.Fatalf("DataPath = %q", got)
	}
	if got := cfg.Repository.StatePath; got != filepath.Join(storageRoot, "data", "backup", "state") {
		t.Fatalf("StatePath = %q", got)
	}
	if got := cfg.Database.Path; got != filepath.Join(storageRoot, "data", "backup", "state", "backup-service.db") {
		t.Fatalf("Database.Path = %q", got)
	}
	if got := cfg.Maintenance.PagePath; got != filepath.Join(storageRoot, "data", "backup", "state", "maintenance.html") {
		t.Fatalf("Maintenance.PagePath = %q", got)
	}
	if got := cfg.Dependencies.MySQLAddress; got != "127.0.0.1:3306" {
		t.Fatalf("MySQLAddress = %q", got)
	}
	if got := cfg.Dependencies.MinIOAddress; got != "127.0.0.1:9000" {
		t.Fatalf("MinIOAddress = %q", got)
	}
}

func TestBackupServiceConfigNormalizeForDevRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	cfg := &BackupServiceConfig{}
	cfg.Storage.Root = "."
	cfg.Storage.NamespacePath = "namespace"
	cfg.Storage.DataPath = "data"
	cfg.Storage.LogsPath = "logs"
	cfg.Storage.MySQLPath = ".local/ai-agent-os/mysql"
	cfg.Storage.MinIOPath = ".local/ai-agent-os/minio"
	cfg.Storage.PodmanStoragePath = ".local/ai-agent-os/podman_storage"
	cfg.Repository.RootPath = "data/backup/repo"
	cfg.Repository.StatePath = "data/backup/state"
	cfg.Repository.StagingPath = "data/backup/staging"
	cfg.Database.Path = "data/backup/state/backup-service.db"
	cfg.Maintenance.MarkerPath = "data/backup/state/maintenance.flag"
	cfg.Maintenance.PagePath = "data/backup/state/maintenance.html"
	cfg.Maintenance.MetadataPath = "data/backup/state/maintenance.json"

	cfg.normalizeForDevRoot(root)

	if got := cfg.Storage.Root; got != root {
		t.Fatalf("Storage.Root = %q, want %q", got, root)
	}
	if got := cfg.Storage.NamespacePath; got != filepath.Join(root, "namespace") {
		t.Fatalf("NamespacePath = %q", got)
	}
	if got := cfg.Repository.RootPath; got != filepath.Join(root, "data", "backup", "repo") {
		t.Fatalf("Repository.RootPath = %q", got)
	}
	if got := cfg.Database.Path; got != filepath.Join(root, "data", "backup", "state", "backup-service.db") {
		t.Fatalf("Database.Path = %q", got)
	}
	if got := cfg.Maintenance.MarkerPath; got != filepath.Join(root, "data", "backup", "state", "maintenance.flag") {
		t.Fatalf("Maintenance.MarkerPath = %q", got)
	}
	if got := cfg.Dependencies.MySQLAddress; got != "127.0.0.1:3306" {
		t.Fatalf("MySQLAddress = %q", got)
	}
	if got := cfg.Dependencies.MinIOAddress; got != "127.0.0.1:9000" {
		t.Fatalf("MinIOAddress = %q", got)
	}
}
