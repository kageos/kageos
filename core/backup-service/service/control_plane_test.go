package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestDeleteSnapshotRemovesArchiveAndRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := newTestBackupServiceConfig(t, tmpDir)
	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	targetDir := filepath.Join(cfg.Storage.NamespacePath, "alice", "demo")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "main.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if _, err := controlPlane.CreateNamespaceSnapshot(ctx, "tester", "snapshot for delete", "alice/demo"); err != nil {
		t.Fatalf("CreateNamespaceSnapshot: %v", err)
	}

	snapshots, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}

	archivePath := snapshots[0].ArchivePath
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("snapshot archive missing before delete: %v", err)
	}

	task, err := controlPlane.DeleteSnapshot(ctx, "tester", "cleanup", snapshots[0].ID)
	if err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}
	if task.Status != backupmodel.TaskStatusSucceeded {
		t.Fatalf("delete task status = %s", task.Status)
	}

	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("expected archive to be removed, err=%v", err)
	}

	snapshotsAfterDelete, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots after delete: %v", err)
	}
	if len(snapshotsAfterDelete) != 0 {
		t.Fatalf("snapshot count after delete = %d, want 0", len(snapshotsAfterDelete))
	}
}

func TestDeleteSnapshotWarnsWhenArchiveAlreadyMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := newTestBackupServiceConfig(t, tmpDir)
	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	targetDir := filepath.Join(cfg.Storage.NamespacePath, "alice", "demo")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "main.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if _, err := controlPlane.CreateNamespaceSnapshot(ctx, "tester", "snapshot for warning delete", "alice/demo"); err != nil {
		t.Fatalf("CreateNamespaceSnapshot: %v", err)
	}

	snapshots, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}

	if err := os.Remove(snapshots[0].ArchivePath); err != nil {
		t.Fatalf("remove archive before delete: %v", err)
	}

	task, err := controlPlane.DeleteSnapshot(ctx, "tester", "cleanup missing archive", snapshots[0].ID)
	if err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}
	if task.Status != backupmodel.TaskStatusWarning {
		t.Fatalf("delete task status = %s, want warning", task.Status)
	}

	snapshotsAfterDelete, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		t.Fatalf("ListSnapshots after delete: %v", err)
	}
	if len(snapshotsAfterDelete) != 0 {
		t.Fatalf("snapshot count after delete = %d, want 0", len(snapshotsAfterDelete))
	}
}

func TestPruneSnapshotsKeepsLatestPreRestore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := newTestBackupServiceConfig(t, tmpDir)
	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	s1 := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceNamespace, backupmodel.SnapshotSourcePreRestore, "older-pre-1")
	s2 := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceNamespace, backupmodel.SnapshotSourcePreRestore, "older-pre-2")
	s3 := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceNamespace, backupmodel.SnapshotSourcePreRestore, "latest-pre")
	manual := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceNamespace, backupmodel.SnapshotSourceManual, "manual")

	task, err := controlPlane.PruneSnapshots(ctx, "tester", "prune old pre-restore", backupmodel.SnapshotResourceNamespace, backupmodel.SnapshotSourcePreRestore, 1)
	if err != nil {
		t.Fatalf("PruneSnapshots: %v", err)
	}
	if task.Status != backupmodel.TaskStatusSucceeded {
		t.Fatalf("prune task status = %s", task.Status)
	}

	snapshots, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 20)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshot count after prune = %d, want 2", len(snapshots))
	}

	remaining := map[int64]SnapshotView{}
	for _, snapshot := range snapshots {
		remaining[snapshot.ID] = snapshot
	}
	if _, ok := remaining[s3.ID]; !ok {
		t.Fatalf("latest pre-restore snapshot %d should remain", s3.ID)
	}
	if _, ok := remaining[manual.ID]; !ok {
		t.Fatalf("manual snapshot %d should remain", manual.ID)
	}
	if _, ok := remaining[s1.ID]; ok {
		t.Fatalf("older snapshot %d should be deleted", s1.ID)
	}
	if _, ok := remaining[s2.ID]; ok {
		t.Fatalf("older snapshot %d should be deleted", s2.ID)
	}
}

func TestPruneSnapshotsWarnsWhenArchiveMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := newTestBackupServiceConfig(t, tmpDir)
	controlPlane, err := NewControlPlane(cfg)
	if err != nil {
		t.Fatalf("NewControlPlane: %v", err)
	}
	defer controlPlane.Close()

	oldest := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceMySQL, backupmodel.SnapshotSourcePreRestore, "old-1")
	middle := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceMySQL, backupmodel.SnapshotSourcePreRestore, "old-2")
	latest := createTestSnapshotRecord(t, ctx, controlPlane, backupmodel.SnapshotResourceMySQL, backupmodel.SnapshotSourcePreRestore, "latest")

	if err := os.Remove(oldest.ArchivePath); err != nil {
		t.Fatalf("remove archive before prune: %v", err)
	}

	task, err := controlPlane.PruneSnapshots(ctx, "tester", "prune with missing archive", backupmodel.SnapshotResourceMySQL, backupmodel.SnapshotSourcePreRestore, 1)
	if err != nil {
		t.Fatalf("PruneSnapshots: %v", err)
	}
	if task.Status != backupmodel.TaskStatusWarning {
		t.Fatalf("prune task status = %s, want warning", task.Status)
	}

	snapshots, err := controlPlane.ListSnapshots(ctx, backupmodel.SnapshotResourceMySQL, 20)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count after prune = %d, want 1", len(snapshots))
	}
	if snapshots[0].ID != latest.ID {
		t.Fatalf("remaining snapshot id = %d, want %d", snapshots[0].ID, latest.ID)
	}
	if _, err := os.Stat(middle.ArchivePath); !os.IsNotExist(err) {
		t.Fatalf("expected middle archive to be removed, err=%v", err)
	}
}

func TestMySQLCommandContextFallsBackToDevContainer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is not portable to windows")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	dockerPath := filepath.Join(binDir, "docker")
	script := "#!/bin/sh\nif [ \"$1\" = \"inspect\" ] && [ \"$2\" = \"ai-agent-os-dev-mysql\" ]; then\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(dockerPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	t.Setenv("APP_ENV", "dev")
	t.Setenv("PATH", binDir)

	controlPlane := &ControlPlane{cfg: &config.BackupServiceConfig{}}
	cmd, err := controlPlane.mysqlCommandContext(ctx, "mysql", []string{"--version"}, []string{"MYSQL_PWD=root"})
	if err != nil {
		t.Fatalf("mysqlCommandContext: %v", err)
	}

	if got := filepath.Base(cmd.Path); got != "docker" {
		t.Fatalf("cmd.Path = %q, want docker", cmd.Path)
	}

	wantArgs := []string{"docker", "exec", "-e", "MYSQL_PWD=root", "ai-agent-os-dev-mysql", "mysql", "--version"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("cmd.Args len = %d, want %d (%v)", len(cmd.Args), len(wantArgs), cmd.Args)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("cmd.Args[%d] = %q, want %q (all=%v)", i, cmd.Args[i], wantArgs[i], cmd.Args)
		}
	}
}

func newTestBackupServiceConfig(t *testing.T, tmpDir string) *config.BackupServiceConfig {
	t.Helper()

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

	return cfg
}

func createTestSnapshotRecord(
	t *testing.T,
	ctx context.Context,
	controlPlane *ControlPlane,
	resourceType string,
	source string,
	note string,
) backupmodel.Snapshot {
	t.Helper()

	archiveDir := filepath.Join(controlPlane.cfg.Repository.RootPath, resourceType)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("mkdir archive dir: %v", err)
	}

	archivePath := filepath.Join(archiveDir, strings.ReplaceAll(note, " ", "-")+".tar.gz")
	if err := os.WriteFile(archivePath, []byte("test-snapshot"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	snapshot := backupmodel.Snapshot{
		ResourceType: resourceType,
		RelativePath: ".",
		Source:       source,
		RequestedBy:  "tester",
		Note:         note,
		ArchivePath:  archivePath,
		ArchiveSize:  int64(len("test-snapshot")),
	}
	if err := controlPlane.store.CreateSnapshot(ctx, &snapshot); err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	return snapshot
}
