package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	backupmodel "github.com/ai-agent-os/ai-agent-os/core/backup-service/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
)

func TestSanitizeRelativePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "", want: "."},
		{input: ".", want: "."},
		{input: "/", want: "."},
		{input: "alice/app", want: "alice/app"},
		{input: "alice//app/../code", want: "alice/code"},
		{input: "../etc", wantErr: true},
	}

	for _, tc := range cases {
		got, err := sanitizeRelativePath(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("sanitizeRelativePath(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("sanitizeRelativePath(%q) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("sanitizeRelativePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFilterMySQLDatabases(t *testing.T) {
	t.Parallel()

	got := filterMySQLDatabases([]string{
		"mysql",
		"information_schema",
		"performance_schema",
		"sys",
		"app_db",
		"agent-server",
		"",
		"  hr-server  ",
	})

	want := []string{"app_db", "agent-server", "hr-server"}
	if len(got) != len(want) {
		t.Fatalf("database count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("database[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNamespaceSnapshotAndRestore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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

	for _, path := range []string{
		cfg.Storage.NamespacePath,
		cfg.Storage.DataPath,
		cfg.Storage.LogsPath,
		cfg.Storage.MySQLPath,
		cfg.Storage.MinIOPath,
		cfg.Storage.PodmanStoragePath,
		cfg.Repository.RootPath,
		cfg.Repository.StatePath,
		cfg.Repository.StagingPath,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	targetDir := filepath.Join(cfg.Storage.NamespacePath, "alice", "demo")
	if err := os.MkdirAll(filepath.Join(targetDir, "code"), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "code", "main.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	snapshotTask, err := controlPlane.CreateNamespaceSnapshot(ctx, "tester", "initial snapshot", "alice/demo")
	if err != nil {
		t.Fatalf("CreateNamespaceSnapshot: %v", err)
	}
	if snapshotTask.Status != backupmodel.TaskStatusSucceeded {
		t.Fatalf("snapshot task status = %s", snapshotTask.Status)
	}

	if err := os.WriteFile(filepath.Join(targetDir, "code", "main.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "code", "extra.txt"), []byte("extra"), 0o644); err != nil {
		t.Fatalf("write extra file: %v", err)
	}

	if _, err := controlPlane.SetMaintenanceMode(ctx, true, "tester", "test restore"); err != nil {
		t.Fatalf("SetMaintenanceMode: %v", err)
	}

	snapshots, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("expected at least one snapshot")
	}

	restoreTask, err := controlPlane.RestoreNamespaceSnapshot(ctx, "tester", "restore snapshot", snapshots[0].ID)
	if err != nil {
		t.Fatalf("RestoreNamespaceSnapshot: %v", err)
	}
	if restoreTask.Status != backupmodel.TaskStatusSucceeded {
		t.Fatalf("restore task status = %s", restoreTask.Status)
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "code", "main.txt"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "v1" {
		t.Fatalf("restored content = %q, want %q", string(content), "v1")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "code", "extra.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected extra file to be removed, got err=%v", err)
	}

	snapshotsAfterRestore, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots after restore: %v", err)
	}
	if len(snapshotsAfterRestore) < 2 {
		t.Fatalf("expected pre-restore snapshot to be created, got %d snapshots", len(snapshotsAfterRestore))
	}
}

func TestSetMaintenanceModeSyncsArtifacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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

	for _, path := range []string{
		cfg.Storage.NamespacePath,
		cfg.Storage.DataPath,
		cfg.Storage.LogsPath,
		cfg.Storage.MySQLPath,
		cfg.Storage.MinIOPath,
		cfg.Storage.PodmanStoragePath,
		cfg.Repository.RootPath,
		cfg.Repository.StatePath,
		cfg.Repository.StagingPath,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	if _, err := controlPlane.SetMaintenanceMode(ctx, true, "tester", `planned <restore>`); err != nil {
		t.Fatalf("SetMaintenanceMode enable: %v", err)
	}

	markerPath := cfg.GetMaintenanceMarkerPath()
	pagePath := cfg.GetMaintenancePagePath()
	metadataPath := cfg.GetMaintenanceMetadataPath()

	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("maintenance marker missing: %v", err)
	}
	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read maintenance page: %v", err)
	}
	pageHTML := string(page)
	if !strings.Contains(pageHTML, "系统维护中") {
		t.Fatalf("maintenance page missing title: %s", pageHTML)
	}
	if !strings.Contains(pageHTML, "planned &lt;restore&gt;") {
		t.Fatalf("maintenance page missing escaped reason: %s", pageHTML)
	}

	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read maintenance metadata: %v", err)
	}
	if !strings.Contains(string(metadata), `"enabled":true`) {
		t.Fatalf("maintenance metadata missing enabled=true: %s", string(metadata))
	}

	if _, err := controlPlane.SetMaintenanceMode(ctx, false, "tester", "done"); err != nil {
		t.Fatalf("SetMaintenanceMode disable: %v", err)
	}

	for _, path := range []string{markerPath, pagePath, metadataPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected maintenance artifact %s to be removed, err=%v", path, err)
		}
	}
}
