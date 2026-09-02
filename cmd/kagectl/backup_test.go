package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupArchiveRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	config := filepath.Join(root, "kage.yaml")
	mustWriteBackupTestFile(t, config, "site:\n  base_url: http://example.test\n")
	dataRoot := filepath.Join(root, "storage")
	for _, name := range restoreDataNames {
		if err := os.MkdirAll(filepath.Join(dataRoot, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteBackupTestFile(t, filepath.Join(dataRoot, "mysql", "ibdata1"), "mysql-data")
	mustWriteBackupTestFile(t, filepath.Join(dataRoot, "minio", "object.bin"), "object-data")
	mustWriteBackupTestFile(t, filepath.Join(dataRoot, "namespace", "app", "main.go"), "package main")
	mustWriteBackupTestFile(t, filepath.Join(dataRoot, "data", "runtime.json"), "{}")
	mustWriteBackupTestFile(t, filepath.Join(dataRoot, "tls", "fullchain.pem"), "certificate")

	archive := filepath.Join(root, "backup.tar.gz")
	manifest := backupManifest{
		Schema: backupSchema, CreatedAt: time.Now().UTC(), MainImage: "kageos:test", AppBaseImage: "kagebase:test", MySQLImage: "mysql:test", MinIOImage: "minio:test",
		StorageRoot: dataRoot, Consistent: true, StoppedStack: true,
	}
	sources := []backupSource{{ArchivePath: "config/kage.yaml", HostPath: config}}
	for _, name := range restoreDataNames {
		sources = append(sources, backupSource{ArchivePath: "data/" + name, HostPath: filepath.Join(dataRoot, name)})
	}
	if err := writeBackupArchive(archive, sources, &manifest); err != nil {
		t.Fatal(err)
	}
	verified, err := verifyBackupArchive(archive)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Schema != backupSchema || len(verified.Entries) == 0 {
		t.Fatalf("unexpected manifest: %#v", verified)
	}

	restored := filepath.Join(root, "restored")
	if err := os.MkdirAll(restored, 0700); err != nil {
		t.Fatal(err)
	}
	if err := extractBackupArchive(archive, restored); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(restored, "data", "namespace", "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package main" {
		t.Fatalf("restored content = %q", got)
	}
}

func TestVerifyBackupRejectsModifiedPayload(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteBackupTestFile(t, filepath.Join(source, "value.txt"), "before")
	manifest := backupManifest{Schema: backupSchema, Consistent: true, StoppedStack: true}
	archive := filepath.Join(root, "backup.tar.gz")
	if err := writeBackupArchive(archive, []backupSource{{ArchivePath: "data/mysql", HostPath: source}}, &manifest); err != nil {
		t.Fatal(err)
	}

	file, err := os.OpenFile(archive, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(-12, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("corruption!!")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyBackupArchive(archive); err == nil {
		t.Fatal("expected corrupted archive verification to fail")
	}
}

func TestVerifyBackupRejectsTraversal(t *testing.T) {
	t.Parallel()
	archive := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	content := []byte("unsafe")
	if err := tw.WriteHeader(&tar.Header{Name: "../escape", Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyBackupArchive(archive); err == nil || !strings.Contains(err.Error(), "unsafe backup archive path") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}

func TestParseBackupAndRestoreFlags(t *testing.T) {
	t.Parallel()
	backup, err := parseBackupFlags([]string{"create", "--output", "/backup"})
	if err != nil {
		t.Fatal(err)
	}
	if backup.Action != "create" || backup.OutputPath != "/backup" {
		t.Fatalf("unexpected backup options: %#v", backup)
	}
	scheduled, err := parseBackupFlags([]string{"scheduled-run"})
	if err != nil || scheduled.Action != "scheduled-run" {
		t.Fatalf("unexpected scheduled backup options: %#v err=%v", scheduled, err)
	}
	verify, err := parseBackupFlags([]string{"verify", "/backup/a.tar.gz"})
	if err != nil || verify.Archive == "" {
		t.Fatalf("unexpected verify options: %#v err=%v", verify, err)
	}
	if _, err := parseRestoreFlags([]string{"/backup/a.tar.gz"}); err == nil {
		t.Fatal("expected restore to require --force or --dry-run")
	}
	restore, err := parseRestoreFlags([]string{"/backup/a.tar.gz", "--dry-run"})
	if err != nil || !restore.DryRun {
		t.Fatalf("unexpected restore options: %#v err=%v", restore, err)
	}
}

func TestMoveAndRestoreRollbackData(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "storage")
	rollback := filepath.Join(t.TempDir(), "rollback")
	if err := os.MkdirAll(rollback, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range restoreDataNames {
		mustWriteBackupTestFile(t, filepath.Join(root, name, "marker"), "old-"+name)
	}
	if err := moveLiveDataToRollback(root, rollback); err != nil {
		t.Fatal(err)
	}
	for _, name := range restoreDataNames {
		mustWriteBackupTestFile(t, filepath.Join(root, name, "marker"), "new-"+name)
	}
	if err := restoreRollbackData(root, rollback); err != nil {
		t.Fatal(err)
	}
	for _, name := range restoreDataNames {
		data, err := os.ReadFile(filepath.Join(root, name, "marker"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "old-"+name {
			t.Fatalf("rollback %s = %q", name, data)
		}
	}
}

func TestMoveLiveDataRollsBackPartialFailure(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "storage")
	rollback := filepath.Join(t.TempDir(), "rollback")
	if err := os.MkdirAll(filepath.Join(rollback, "minio"), 0700); err != nil {
		t.Fatal(err)
	}
	mustWriteBackupTestFile(t, filepath.Join(rollback, "minio", "collision"), "occupied")
	for _, name := range restoreDataNames {
		mustWriteBackupTestFile(t, filepath.Join(root, name, "marker"), "old-"+name)
	}
	if err := moveLiveDataToRollback(root, rollback); err == nil {
		t.Fatal("expected rollback destination collision to fail")
	}
	for _, name := range restoreDataNames {
		data, err := os.ReadFile(filepath.Join(root, name, "marker"))
		if err != nil {
			t.Fatalf("live %s was not restored after partial move: %v", name, err)
		}
		if string(data) != "old-"+name {
			t.Fatalf("live %s = %q", name, data)
		}
	}
}

func mustWriteBackupTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
