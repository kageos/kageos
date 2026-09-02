package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/systembackup"
	"gopkg.in/yaml.v3"
)

const (
	backupSchema       = "kageos.backup.v1"
	backupArchiveExt   = ".tar.gz"
	backupManifestPath = "manifest.json"
)

type backupManifest struct {
	Schema       string        `json:"schema"`
	CreatedAt    time.Time     `json:"created_at"`
	MainImage    string        `json:"main_image"`
	AppBaseImage string        `json:"app_base_image"`
	MySQLImage   string        `json:"mysql_image"`
	MinIOImage   string        `json:"minio_image"`
	StorageRoot  string        `json:"storage_root"`
	Consistent   bool          `json:"consistent"`
	StoppedStack bool          `json:"stopped_stack"`
	Entries      []backupEntry `json:"entries"`
}

type backupEntry struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       int64  `json:"size,omitempty"`
	Mode       uint32 `json:"mode,omitempty"`
	UID        int    `json:"uid,omitempty"`
	GID        int    `json:"gid,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	LinkTarget string `json:"link_target,omitempty"`
}

type backupSource struct {
	ArchivePath string
	HostPath    string
}

type observedBackupEntry struct {
	Type       string
	Size       int64
	Mode       uint32
	UID        int
	GID        int
	SHA256     string
	LinkTarget string
}

func cmdBackup(paths Paths, args []string) error {
	opts, err := parseBackupFlags(args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return fmt.Errorf("backup is currently available only in prod mode")
	}
	switch opts.Action {
	case "create":
		return createProductionBackup(paths, opts)
	case "list":
		return listProductionBackups(paths)
	case "verify":
		manifest, err := verifyBackupArchive(resolveUserPath(opts.Archive))
		if err != nil {
			return err
		}
		fmt.Printf("backup verified: %s\n", resolveUserPath(opts.Archive))
		fmt.Printf("created: %s\n", manifest.CreatedAt.Format(time.RFC3339))
		fmt.Printf("entries: %d\n", len(manifest.Entries))
		return nil
	case "scheduled-run":
		return runScheduledProductionBackup(paths, time.Now())
	default:
		return fmt.Errorf("unsupported backup action %q", opts.Action)
	}
}

func runScheduledProductionBackup(paths Paths, now time.Time) error {
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	location, err := time.LoadLocation(rt.Timezone)
	if err != nil {
		return fmt.Errorf("load backup timezone %q: %w", rt.Timezone, err)
	}
	now = now.In(location)
	dir := filepath.Join(rt.Storage.Root, "data", "system-backup")
	state, err := systembackup.LoadState(dir)
	if err != nil {
		return err
	}
	state.AgentLastSeenAt = now.UTC().Format(time.RFC3339)
	if err := systembackup.SaveState(dir, state); err != nil {
		return err
	}
	cfg, err := systembackup.LoadConfig(dir, rt.Secrets.JWTSecret)
	if err != nil {
		return err
	}
	due, trigger := systembackup.Due(cfg, state, now)
	if !due {
		return nil
	}

	lockPath := filepath.Join(dir, "agent.lock")
	if info, statErr := os.Stat(lockPath); statErr == nil && now.Sub(info.ModTime()) > 12*time.Hour {
		_ = os.Remove(lockPath)
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("acquire backup agent lock: %w", err)
	}
	_ = lock.Close()
	defer os.Remove(lockPath)

	if trigger == "manual" {
		cfg.LastRunNowProcessedAt = cfg.RunNowRequestedAt
		if err := systembackup.SaveConfig(dir, rt.Secrets.JWTSecret, cfg); err != nil {
			return err
		}
	}
	record := systembackup.Record{ID: now.UTC().Format("20060102T150405.000000000Z"), TriggeredBy: trigger, Status: "running", StartedAt: now.UTC().Format(time.RFC3339)}
	state.Running = true
	state.Records = append([]systembackup.Record{record}, state.Records...)
	if err := systembackup.SaveState(dir, state); err != nil {
		return err
	}

	finish := func(runErr error) error {
		latest, loadErr := systembackup.LoadState(dir)
		if loadErr != nil {
			return loadErr
		}
		latest.Running = false
		latest.AgentLastSeenAt = time.Now().UTC().Format(time.RFC3339)
		for i := range latest.Records {
			if latest.Records[i].ID != record.ID {
				continue
			}
			latest.Records[i].FinishedAt = time.Now().UTC().Format(time.RFC3339)
			if runErr != nil {
				latest.Records[i].Status = "failed"
				latest.Records[i].ErrorMessage = runErr.Error()
			} else {
				latest.Records[i].Status = "succeeded"
			}
			break
		}
		if saveErr := systembackup.SaveState(dir, latest); saveErr != nil {
			return saveErr
		}
		return runErr
	}

	archivePath, err := resolveBackupOutputPath(paths, "", now)
	if err != nil {
		return finish(err)
	}
	if err := createProductionBackup(paths, backupOptions{Action: "create", OutputPath: archivePath}); err != nil {
		return finish(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()
	objectKey, etag, size, checksum, err := systembackup.Upload(ctx, cfg, archivePath)
	if err != nil {
		return finish(err)
	}
	latest, err := systembackup.LoadState(dir)
	if err != nil {
		return finish(err)
	}
	for i := range latest.Records {
		if latest.Records[i].ID == record.ID {
			latest.Records[i].ArchiveName = filepath.Base(archivePath)
			latest.Records[i].SizeBytes = size
			latest.Records[i].SHA256 = checksum
			latest.Records[i].Bucket = cfg.Bucket
			latest.Records[i].ObjectKey = objectKey
			latest.Records[i].ETag = etag
			break
		}
	}
	if err := systembackup.SaveState(dir, latest); err != nil {
		return finish(err)
	}
	if err := systembackup.PruneLocal(filepath.Dir(archivePath), cfg.KeepLocal); err != nil {
		return finish(fmt.Errorf("backup uploaded but local cleanup failed: %w", err))
	}
	if err := systembackup.PruneRemote(ctx, cfg, time.Now()); err != nil {
		return finish(fmt.Errorf("backup uploaded but remote cleanup failed: %w", err))
	}
	return finish(nil)
}

func createProductionBackup(paths Paths, opts backupOptions) error {
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateBackupRuntime(rt); err != nil {
		return err
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}

	archivePath, err := resolveBackupOutputPath(paths, opts.OutputPath, time.Now())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	if _, err := os.Lstat(archivePath); err == nil {
		return fmt.Errorf("refuse to overwrite existing backup: %s", archivePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if samePathOrInside(archivePath, rt.Storage.Root) {
		return fmt.Errorf("backup output must be outside production storage root %s", rt.Storage.Root)
	}

	fmt.Println("==> stop kageos stack for a consistent physical backup")
	if err := runComposeCapture(rt.Paths.GeneratedDir, "stop"); err != nil {
		return fmt.Errorf("stop stack before backup: %w", err)
	}
	stackStopped := true
	defer func() {
		if stackStopped {
			fmt.Println("==> restart kageos stack after backup attempt")
			if startErr := runComposeCapture(rt.Paths.GeneratedDir, "start"); startErr != nil {
				fmt.Fprintf(os.Stderr, "ERROR: restart stack after backup: %v\n", startErr)
			}
		}
	}()

	sources := productionBackupSources(paths, rt)
	manifest := backupManifest{
		Schema: backupSchema, CreatedAt: time.Now().UTC(), MainImage: rt.Images.Main, AppBaseImage: rt.Images.AppBase, MySQLImage: rt.Images.MySQL,
		MinIOImage: rt.Images.MinIO, StorageRoot: rt.Storage.Root, Consistent: true, StoppedStack: true,
	}
	fmt.Printf("==> create backup archive: %s\n", archivePath)
	if err := writeBackupArchive(archivePath, sources, &manifest); err != nil {
		return err
	}
	if _, err := verifyBackupArchive(archivePath); err != nil {
		return fmt.Errorf("verify newly created backup: %w", err)
	}
	if err := writeBackupArchiveChecksum(archivePath); err != nil {
		return err
	}

	fmt.Println("==> restart kageos stack")
	if err := runComposeCapture(rt.Paths.GeneratedDir, "start"); err != nil {
		return fmt.Errorf("backup created but stack restart failed: %w", err)
	}
	stackStopped = false
	if err := waitLayerChecks("backup restart verify", verifyLayerChecks(rt), defaultUpVerifyTimeout, defaultUpVerifyInterval); err != nil {
		return fmt.Errorf("backup created but restarted instance did not become healthy: %w", err)
	}
	fmt.Printf("backup created and verified: %s\n", archivePath)
	fmt.Printf("checksum: %s.sha256\n", archivePath)
	fmt.Println("IMPORTANT: keep a copy on another disk or server; a same-host backup does not protect against disk loss")
	return nil
}

func validateBackupRuntime(rt RuntimeConfig) error {
	if rt.MySQL.Mode != "bundled" || !rt.IncludeMySQL {
		return fmt.Errorf("backup currently requires mysql.mode=bundled; use the external database provider backup for mysql.mode=%s", rt.MySQL.Mode)
	}
	if rt.MinIO.Mode != "bundled" || !rt.IncludeMinIO {
		return fmt.Errorf("backup currently requires minio.mode=bundled; use the external object storage backup for minio.mode=%s", rt.MinIO.Mode)
	}
	return nil
}

func productionBackupSources(paths Paths, rt RuntimeConfig) []backupSource {
	return []backupSource{
		{ArchivePath: "config/kage.yaml", HostPath: paths.ConfigPath},
		{ArchivePath: "data/mysql", HostPath: filepath.Join(rt.Storage.Root, "mysql")},
		{ArchivePath: "data/minio", HostPath: filepath.Join(rt.Storage.Root, "minio")},
		{ArchivePath: "data/namespace", HostPath: filepath.Join(rt.Storage.Root, "namespace")},
		{ArchivePath: "data/data", HostPath: filepath.Join(rt.Storage.Root, "data")},
		{ArchivePath: "data/tls", HostPath: filepath.Join(rt.Storage.Root, "tls")},
	}
}

func resolveBackupOutputPath(paths Paths, requested string, now time.Time) (string, error) {
	name := fmt.Sprintf("kageos-backup-%s%s", now.UTC().Format("20060102T150405Z"), backupArchiveExt)
	if strings.TrimSpace(requested) == "" {
		return filepath.Join(paths.ProdDir, "backups", name), nil
	}
	requested = resolveUserPath(requested)
	if strings.HasSuffix(requested, backupArchiveExt) {
		return requested, nil
	}
	if info, err := os.Stat(requested); err == nil && !info.IsDir() {
		return "", fmt.Errorf("backup output is not a directory: %s", requested)
	}
	return filepath.Join(requested, name), nil
}

func resolveUserPath(path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func samePathOrInside(path, root string) bool {
	path, root = filepath.Clean(path), filepath.Clean(root)
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeBackupArchive(path string, sources []backupSource, manifest *backupManifest) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kageos-backup-*.partial")
	if err != nil {
		return fmt.Errorf("create temporary backup: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)
	for _, source := range sources {
		if err := addBackupSource(tw, source, manifest); err != nil {
			tw.Close()
			gz.Close()
			tmp.Close()
			return err
		}
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil {
		err = writeTarBytes(tw, backupManifestPath, manifestBytes, 0600)
	}
	if closeErr := tw.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gz.Close(); err == nil {
		err = closeErr
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("finalize backup archive: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish backup archive: %w", err)
	}
	return nil
}

func addBackupSource(tw *tar.Writer, source backupSource, manifest *backupManifest) error {
	rootInfo, err := os.Lstat(source.HostPath)
	if err != nil {
		return fmt.Errorf("backup source %s: %w", source.HostPath, err)
	}
	return filepath.Walk(source.HostPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source.HostPath, path)
		if err != nil {
			return err
		}
		archiveName := source.ArchivePath
		if rel != "." {
			archiveName = filepath.ToSlash(filepath.Join(source.ArchivePath, rel))
		} else if !rootInfo.IsDir() {
			archiveName = source.ArchivePath
		}
		return writeBackupEntry(tw, path, archiveName, info, manifest)
	})
}

func writeBackupEntry(tw *tar.Writer, hostPath, archiveName string, info os.FileInfo, manifest *backupManifest) error {
	archiveName = filepath.ToSlash(filepath.Clean(archiveName))
	entry := backupEntry{Path: archiveName, Mode: uint32(info.Mode().Perm())}
	linkTarget := ""
	if info.Mode()&os.ModeSymlink != 0 {
		var err error
		linkTarget, err = os.Readlink(hostPath)
		if err != nil {
			return err
		}
		entry.Type, entry.LinkTarget = "symlink", linkTarget
	} else if info.IsDir() {
		entry.Type = "dir"
	} else if info.Mode().IsRegular() {
		entry.Type, entry.Size = "file", info.Size()
	} else {
		return fmt.Errorf("backup source contains unsupported special file: %s", hostPath)
	}

	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return err
	}
	header.Name = archiveName
	header.Uname, header.Gname = "", ""
	entry.UID, entry.GID = header.Uid, header.Gid
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if entry.Type == "file" {
		file, err := os.Open(hostPath)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(tw, h), file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		entry.SHA256 = hex.EncodeToString(h.Sum(nil))
	}
	manifest.Entries = append(manifest.Entries, entry)
	return nil
}

func writeTarBytes(tw *tar.Writer, name string, data []byte, mode int64) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), ModTime: time.Now().UTC(), Typeflag: tar.TypeReg}); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func verifyBackupArchive(path string) (*backupManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open backup archive: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open backup gzip: %w", err)
	}
	defer gz.Close()

	observed := map[string]observedBackupEntry{}
	var manifest backupManifest
	foundManifest := false
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read backup archive: %w", err)
		}
		name, err := validateArchivePath(header.Name)
		if err != nil {
			return nil, err
		}
		if name == backupManifestPath {
			if foundManifest {
				return nil, fmt.Errorf("backup contains duplicate manifest")
			}
			data, err := io.ReadAll(io.LimitReader(tr, 16<<20))
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("parse backup manifest: %w", err)
			}
			foundManifest = true
			continue
		}
		if _, exists := observed[name]; exists {
			return nil, fmt.Errorf("backup contains duplicate path %s", name)
		}
		entry := observedBackupEntry{Mode: uint32(header.Mode & 0777), UID: header.Uid, GID: header.Gid, Size: header.Size}
		switch header.Typeflag {
		case tar.TypeDir:
			entry.Type = "dir"
		case tar.TypeReg, tar.TypeRegA:
			entry.Type = "file"
			h := sha256.New()
			if _, err := io.Copy(h, tr); err != nil {
				return nil, err
			}
			entry.SHA256 = hex.EncodeToString(h.Sum(nil))
		case tar.TypeSymlink:
			entry.Type, entry.LinkTarget = "symlink", header.Linkname
		default:
			return nil, fmt.Errorf("backup contains unsupported tar entry %s", name)
		}
		observed[name] = entry
	}
	if !foundManifest {
		return nil, fmt.Errorf("backup manifest is missing")
	}
	if manifest.Schema != backupSchema || !manifest.Consistent || !manifest.StoppedStack {
		return nil, fmt.Errorf("unsupported or inconsistent backup manifest: schema=%q", manifest.Schema)
	}
	if err := compareBackupManifest(manifest.Entries, observed); err != nil {
		return nil, err
	}
	for _, required := range []string{"config/kage.yaml", "data/mysql", "data/minio", "data/namespace", "data/data", "data/tls"} {
		if _, ok := observed[required]; !ok {
			return nil, fmt.Errorf("backup is missing required entry %s", required)
		}
	}
	return &manifest, nil
}

func compareBackupManifest(entries []backupEntry, observed map[string]observedBackupEntry) error {
	if len(entries) != len(observed) {
		return fmt.Errorf("backup entry count mismatch: manifest=%d archive=%d", len(entries), len(observed))
	}
	seen := map[string]struct{}{}
	for _, expected := range entries {
		if _, ok := seen[expected.Path]; ok {
			return fmt.Errorf("manifest contains duplicate path %s", expected.Path)
		}
		seen[expected.Path] = struct{}{}
		actual, ok := observed[expected.Path]
		if !ok {
			return fmt.Errorf("backup is missing manifest entry %s", expected.Path)
		}
		if actual.Type != expected.Type || actual.Size != expected.Size || actual.Mode != expected.Mode || actual.UID != expected.UID || actual.GID != expected.GID || actual.SHA256 != expected.SHA256 || actual.LinkTarget != expected.LinkTarget {
			return fmt.Errorf("backup entry verification failed: %s", expected.Path)
		}
	}
	return nil
}

func validateArchivePath(name string) (string, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	clean := filepath.ToSlash(filepath.Clean(name))
	if name == "" || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || clean != name {
		return "", fmt.Errorf("unsafe backup archive path %q", name)
	}
	return clean, nil
}

func writeBackupArchiveChecksum(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, file)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(h.Sum(nil)), filepath.Base(path))
	return os.WriteFile(path+".sha256", []byte(line), 0600)
}

func listProductionBackups(paths Paths) error {
	dir := filepath.Join(paths.ProdDir, "backups")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("no backups found in %s\n", dir)
		return nil
	}
	if err != nil {
		return err
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), backupArchiveExt) {
			names = append(names, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(names)
	for _, name := range names {
		info, err := os.Stat(name)
		if err != nil {
			return err
		}
		fmt.Printf("%s\t%d bytes\n", name, info.Size())
	}
	if len(names) == 0 {
		fmt.Printf("no backups found in %s\n", dir)
	}
	return nil
}

func cmdRestore(paths Paths, args []string) error {
	opts, err := parseRestoreFlags(args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return fmt.Errorf("restore is currently available only in prod mode")
	}
	archivePath := resolveUserPath(opts.Archive)
	manifest, err := verifyBackupArchive(archivePath)
	if err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateBackupRuntime(rt); err != nil {
		return err
	}
	if manifest.MainImage != rt.Images.Main || manifest.AppBaseImage != rt.Images.AppBase || manifest.MySQLImage != rt.Images.MySQL || manifest.MinIOImage != rt.Images.MinIO {
		return fmt.Errorf("backup image versions do not match current config: main %q/%q, app-base %q/%q, mysql %q/%q, minio %q/%q", manifest.MainImage, rt.Images.Main, manifest.AppBaseImage, rt.Images.AppBase, manifest.MySQLImage, rt.Images.MySQL, manifest.MinIOImage, rt.Images.MinIO)
	}
	printRestorePlan(archivePath, rt, manifest)
	if opts.DryRun {
		fmt.Println("restore dry-run passed; no files were changed")
		return nil
	}
	return executeProductionRestore(paths, rt, archivePath, opts)
}

func printRestorePlan(archive string, rt RuntimeConfig, manifest *backupManifest) {
	fmt.Printf("backup: %s\n", archive)
	fmt.Printf("created: %s\n", manifest.CreatedAt.Format(time.RFC3339))
	fmt.Printf("target storage: %s\n", rt.Storage.Root)
	fmt.Println("restore replaces: mysql, minio, namespace, data, tls, and the private production config")
}

func executeProductionRestore(paths Paths, rt RuntimeConfig, archivePath string, opts restoreOptions) error {
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	parent := filepath.Dir(filepath.Clean(rt.Storage.Root))
	staging, err := os.MkdirTemp(parent, ".kageos-restore-staging-*")
	if err != nil {
		return fmt.Errorf("create restore staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	if err := extractBackupArchive(archivePath, staging); err != nil {
		return err
	}

	backedConfigPath := filepath.Join(staging, "config", "kage.yaml")
	backedConfigData, err := os.ReadFile(backedConfigPath)
	if err != nil {
		return err
	}
	var backedConfig Config
	if err := yaml.Unmarshal(backedConfigData, &backedConfig); err != nil {
		return fmt.Errorf("parse restored config: %w", err)
	}
	backedConfig.Storage.Root = rt.Storage.Root
	backedConfigData, err = yaml.Marshal(backedConfig)
	if err != nil {
		return err
	}

	oldConfig, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		return err
	}
	rollback := filepath.Join(parent, fmt.Sprintf(".%s.restore-rollback-%s", filepath.Base(rt.Storage.Root), time.Now().UTC().Format("20060102T150405Z")))
	if err := os.MkdirAll(rollback, 0700); err != nil {
		return err
	}

	fmt.Println("==> stop kageos stack before restore")
	if err := runComposeCapture(rt.Paths.GeneratedDir, "stop"); err != nil {
		return fmt.Errorf("stop stack before restore: %w", err)
	}
	replaced := false
	rollbackOnFailure := func(cause error) error {
		if !replaced {
			_ = runComposeCapture(rt.Paths.GeneratedDir, "start")
			return cause
		}
		_ = runComposeCapture(rt.Paths.GeneratedDir, "down")
		if rollbackErr := restoreRollbackData(rt.Storage.Root, rollback); rollbackErr != nil {
			return fmt.Errorf("%v; automatic rollback also failed: %w; rollback data: %s", cause, rollbackErr, rollback)
		}
		_ = os.WriteFile(paths.ConfigPath, oldConfig, 0600)
		_ = cmdRender(paths)
		if startErr := cmdUp(paths, []string{"--no-build"}); startErr != nil {
			return fmt.Errorf("%v; old data restored but stack restart failed: %w", cause, startErr)
		}
		return fmt.Errorf("%v; original data was restored automatically", cause)
	}

	if err := moveLiveDataToRollback(rt.Storage.Root, rollback); err != nil {
		return rollbackOnFailure(err)
	}
	replaced = true
	if err := moveRestoredDataIntoPlace(staging, rt.Storage.Root); err != nil {
		return rollbackOnFailure(err)
	}
	if err := os.WriteFile(paths.ConfigPath, backedConfigData, 0600); err != nil {
		return rollbackOnFailure(err)
	}
	if err := cmdRender(paths); err != nil {
		return rollbackOnFailure(err)
	}
	fmt.Println("==> start and verify restored kageos instance")
	if err := cmdUp(paths, []string{"--no-build"}); err != nil {
		return rollbackOnFailure(err)
	}
	if opts.KeepRollback {
		fmt.Printf("restore completed; rollback data retained at %s\n", rollback)
	} else if err := os.RemoveAll(rollback); err != nil {
		return fmt.Errorf("restore succeeded but rollback cleanup failed: %w", err)
	}
	fmt.Println("restore completed and instance verification passed")
	return nil
}

func extractBackupArchive(path, destination string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	type pendingLink struct{ path, target string }
	var links []pendingLink
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name, err := validateArchivePath(header.Name)
		if err != nil {
			return err
		}
		if name == backupManifestPath {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !samePathOrInside(target, destination) {
			return fmt.Errorf("archive path escapes restore directory: %s", name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)&0777); err != nil {
				return err
			}
			if err := restoreOwnership(target, header, false); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if err := restoreOwnership(target, header, false); err != nil {
				return err
			}
		case tar.TypeSymlink:
			links = append(links, pendingLink{path: target, target: header.Linkname})
		default:
			return fmt.Errorf("unsupported restore entry: %s", name)
		}
	}
	for _, link := range links {
		if filepath.IsAbs(link.target) {
			return fmt.Errorf("backup contains absolute symlink target %q", link.target)
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(link.path), link.target))
		if !samePathOrInside(resolved, destination) {
			return fmt.Errorf("backup symlink escapes restore directory: %s", link.path)
		}
		if err := os.MkdirAll(filepath.Dir(link.path), 0700); err != nil {
			return err
		}
		if err := os.Symlink(link.target, link.path); err != nil {
			return err
		}
	}
	return nil
}

var restoreDataNames = []string{"mysql", "minio", "namespace", "data", "tls"}

func moveLiveDataToRollback(storageRoot, rollback string) error {
	moved := make([]string, 0, len(restoreDataNames))
	for _, name := range restoreDataNames {
		source := filepath.Join(storageRoot, name)
		if _, err := os.Lstat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Rename(source, filepath.Join(rollback, name)); err != nil {
			for i := len(moved) - 1; i >= 0; i-- {
				movedName := moved[i]
				if rollbackErr := os.Rename(filepath.Join(rollback, movedName), filepath.Join(storageRoot, movedName)); rollbackErr != nil {
					return fmt.Errorf("move live %s to rollback: %v; revert %s failed: %w", name, err, movedName, rollbackErr)
				}
			}
			return fmt.Errorf("move live %s to rollback: %w", name, err)
		}
		moved = append(moved, name)
	}
	return nil
}

func restoreOwnership(path string, header *tar.Header, symlink bool) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if symlink {
		return os.Lchown(path, header.Uid, header.Gid)
	}
	return os.Chown(path, header.Uid, header.Gid)
}

func moveRestoredDataIntoPlace(staging, storageRoot string) error {
	if err := os.MkdirAll(storageRoot, 0755); err != nil {
		return err
	}
	for _, name := range restoreDataNames {
		source := filepath.Join(staging, "data", name)
		if _, err := os.Lstat(source); err != nil {
			return fmt.Errorf("restored data %s missing: %w", name, err)
		}
		if err := os.Rename(source, filepath.Join(storageRoot, name)); err != nil {
			return fmt.Errorf("install restored %s: %w", name, err)
		}
	}
	return nil
}

func restoreRollbackData(storageRoot, rollback string) error {
	for _, name := range restoreDataNames {
		_ = os.RemoveAll(filepath.Join(storageRoot, name))
		source := filepath.Join(rollback, name)
		if _, err := os.Lstat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Rename(source, filepath.Join(storageRoot, name)); err != nil {
			return err
		}
	}
	return nil
}
