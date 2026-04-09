package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	backupmodel "github.com/ai-agent-os/ai-agent-os/core/backup-service/model"
	"github.com/ai-agent-os/ai-agent-os/core/backup-service/store"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	ErrTaskAlreadyRunning      = errors.New("another backup task is already running")
	ErrMaintenanceModeRequired = errors.New("maintenance mode must be enabled before restore")
	ErrInvalidRelativePath     = errors.New("invalid relative path")
)

type ControlPlane struct {
	cfg    *config.BackupServiceConfig
	store  *store.Store
	taskMu sync.Mutex
}

type Status struct {
	Service         string               `json:"service"`
	StorageRoot     string               `json:"storage_root"`
	Repository      RepositoryState      `json:"repository"`
	SystemState     SystemStateView      `json:"system_state"`
	Paths           map[string]PathCheck `json:"paths"`
	RecentTasks     []TaskView           `json:"recent_tasks"`
	RecentSnapshots []SnapshotView       `json:"recent_snapshots"`
}

type RepositoryState struct {
	RootPath                string `json:"root_path"`
	StatePath               string `json:"state_path"`
	StagingPath             string `json:"staging_path"`
	DatabasePath            string `json:"database_path"`
	MaintenanceMarkerPath   string `json:"maintenance_marker_path"`
	MaintenancePagePath     string `json:"maintenance_page_path"`
	MaintenanceMetadataPath string `json:"maintenance_metadata_path"`
}

type SystemStateView struct {
	MaintenanceMode      bool       `json:"maintenance_mode"`
	MaintenanceReason    string     `json:"maintenance_reason"`
	ActiveTaskID         *int64     `json:"active_task_id,omitempty"`
	LastPrecheckAt       *time.Time `json:"last_precheck_at,omitempty"`
	LastPrecheckTaskID   *int64     `json:"last_precheck_task_id,omitempty"`
	MaintenanceUpdatedAt *time.Time `json:"maintenance_updated_at,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type TaskView struct {
	ID           int64       `json:"id"`
	Type         string      `json:"type"`
	Status       string      `json:"status"`
	RequestedBy  string      `json:"requested_by"`
	Note         string      `json:"note"`
	Summary      string      `json:"summary"`
	ErrorMessage string      `json:"error_message"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	FinishedAt   *time.Time  `json:"finished_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Detail       interface{} `json:"detail,omitempty"`
}

type SnapshotView struct {
	ID             int64       `json:"id"`
	ResourceType   string      `json:"resource_type"`
	RelativePath   string      `json:"relative_path"`
	Source         string      `json:"source"`
	RequestedBy    string      `json:"requested_by"`
	Note           string      `json:"note"`
	ArchivePath    string      `json:"archive_path"`
	ArchiveSize    int64       `json:"archive_size"`
	FileCount      int64       `json:"file_count"`
	DirectoryCount int64       `json:"directory_count"`
	Metadata       interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type PathCheck struct {
	Path             string `json:"path"`
	Exists           bool   `json:"exists"`
	IsDir            bool   `json:"is_dir"`
	WritableExpected bool   `json:"writable_expected"`
	Writable         bool   `json:"writable"`
	Error            string `json:"error,omitempty"`
}

type DependencyCheck struct {
	Address   string `json:"address"`
	Reachable bool   `json:"reachable"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ToolCheck struct {
	Binary string `json:"binary"`
	Path   string `json:"path,omitempty"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

type FilesystemCheck struct {
	Path           string `json:"path"`
	TotalBytes     uint64 `json:"total_bytes"`
	FreeBytes      uint64 `json:"free_bytes"`
	AvailableBytes uint64 `json:"available_bytes"`
	Error          string `json:"error,omitempty"`
}

type PrecheckResult struct {
	Ready        bool                       `json:"ready"`
	GeneratedAt  time.Time                  `json:"generated_at"`
	Paths        map[string]PathCheck       `json:"paths"`
	Dependencies map[string]DependencyCheck `json:"dependencies"`
	Tooling      map[string]ToolCheck       `json:"tooling"`
	Filesystems  map[string]FilesystemCheck `json:"filesystems"`
	Issues       []string                   `json:"issues"`
	Warnings     []string                   `json:"warnings"`
}

type NamespaceSnapshotResult struct {
	Snapshot SnapshotView `json:"snapshot"`
}

type NamespaceRestoreResult struct {
	RestoredSnapshot   SnapshotView  `json:"restored_snapshot"`
	PreRestoreSnapshot *SnapshotView `json:"pre_restore_snapshot,omitempty"`
	RestoredPath       string        `json:"restored_path"`
}

type MySQLSnapshotResult struct {
	Snapshot  SnapshotView `json:"snapshot"`
	Databases []string     `json:"databases"`
}

type MySQLRestoreResult struct {
	RestoredSnapshot   SnapshotView  `json:"restored_snapshot"`
	PreRestoreSnapshot *SnapshotView `json:"pre_restore_snapshot,omitempty"`
	Databases          []string      `json:"databases"`
}

type MinIOSnapshotResult struct {
	Snapshot SnapshotView `json:"snapshot"`
	Buckets  []string     `json:"buckets"`
	Objects  int64        `json:"objects"`
}

type MinIORestoreResult struct {
	RestoredSnapshot   SnapshotView  `json:"restored_snapshot"`
	PreRestoreSnapshot *SnapshotView `json:"pre_restore_snapshot,omitempty"`
	Buckets            []string      `json:"buckets"`
	Objects            int64         `json:"objects"`
}

type SnapshotDeleteResult struct {
	DeletedSnapshot SnapshotView `json:"deleted_snapshot"`
	ArchiveDeleted  bool         `json:"archive_deleted"`
	ArchiveMissing  bool         `json:"archive_missing"`
}

type minIOSnapshotManifest struct {
	Buckets []minIOBucketManifest `json:"buckets"`
}

type minIOBucketManifest struct {
	Name    string                `json:"name"`
	Objects []minIOObjectManifest `json:"objects"`
}

type minIOObjectManifest struct {
	Key          string            `json:"key"`
	BlobPath     string            `json:"blob_path"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type,omitempty"`
	UserMetadata map[string]string `json:"user_metadata,omitempty"`
}

type maintenanceStateFile struct {
	Enabled   bool       `json:"enabled"`
	Reason    string     `json:"reason,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type snapshotArchiveStats struct {
	FileCount      int64
	DirectoryCount int64
	ArchiveSize    int64
}

func NewControlPlane(cfg *config.BackupServiceConfig) (*ControlPlane, error) {
	s, err := store.New(cfg.GetDatabasePath())
	if err != nil {
		return nil, err
	}

	controlPlane := &ControlPlane{
		cfg:   cfg,
		store: s,
	}

	state, err := s.GetSystemState(context.Background())
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := controlPlane.syncMaintenanceArtifacts(state); err != nil {
		_ = s.Close()
		return nil, err
	}

	return controlPlane, nil
}

func (c *ControlPlane) Close() error {
	return c.store.Close()
}

func (c *ControlPlane) GetStatus(ctx context.Context) (*Status, error) {
	state, err := c.store.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := c.store.ListTasks(ctx, 10)
	if err != nil {
		return nil, err
	}
	snapshots, err := c.store.ListSnapshots(ctx, backupmodel.SnapshotResourceNamespace, 10)
	if err != nil {
		return nil, err
	}

	return &Status{
		Service:     "backup-service",
		StorageRoot: c.cfg.Storage.Root,
		Repository: RepositoryState{
			RootPath:                c.cfg.Repository.RootPath,
			StatePath:               c.cfg.Repository.StatePath,
			StagingPath:             c.cfg.Repository.StagingPath,
			DatabasePath:            c.cfg.GetDatabasePath(),
			MaintenanceMarkerPath:   c.cfg.GetMaintenanceMarkerPath(),
			MaintenancePagePath:     c.cfg.GetMaintenancePagePath(),
			MaintenanceMetadataPath: c.cfg.GetMaintenanceMetadataPath(),
		},
		SystemState:     toSystemStateView(state),
		Paths:           c.statusPathReport(),
		RecentTasks:     toTaskViews(tasks),
		RecentSnapshots: toSnapshotViews(snapshots),
	}, nil
}

func (c *ControlPlane) ListTasks(ctx context.Context, limit int) ([]TaskView, error) {
	tasks, err := c.store.ListTasks(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toTaskViews(tasks), nil
}

func (c *ControlPlane) GetTask(ctx context.Context, id int64) (*TaskView, error) {
	task, err := c.store.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) ListSnapshots(ctx context.Context, resourceType string, limit int) ([]SnapshotView, error) {
	snapshots, err := c.store.ListSnapshots(ctx, resourceType, limit)
	if err != nil {
		return nil, err
	}
	return toSnapshotViews(snapshots), nil
}

func (c *ControlPlane) GetSnapshot(ctx context.Context, id int64) (*SnapshotView, error) {
	snapshot, err := c.store.GetSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	view := toSnapshotView(snapshot)
	return &view, nil
}

func (c *ControlPlane) SetMaintenanceMode(ctx context.Context, enabled bool, _ string, reason string) (*SystemStateView, error) {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()

	now := time.Now()
	var previousState backupmodel.SystemState
	currentState, err := c.store.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}
	previousState = *currentState

	state, err := c.store.UpdateSystemState(ctx, func(state *backupmodel.SystemState) error {
		state.MaintenanceMode = enabled
		state.MaintenanceReason = strings.TrimSpace(reason)
		state.MaintenanceUpdatedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := c.syncMaintenanceArtifacts(state); err != nil {
		_, rollbackErr := c.store.UpdateSystemState(ctx, func(state *backupmodel.SystemState) error {
			state.MaintenanceMode = previousState.MaintenanceMode
			state.MaintenanceReason = previousState.MaintenanceReason
			state.MaintenanceUpdatedAt = previousState.MaintenanceUpdatedAt
			return nil
		})
		if rollbackErr != nil {
			return nil, fmt.Errorf("sync maintenance artifacts failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	view := toSystemStateView(state)
	return &view, nil
}

func (c *ControlPlane) RunPrecheck(ctx context.Context, requestedBy string, note string) (*TaskView, error) {
	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypePrecheck, requestedBy, note, "环境预检执行中", func(task *backupmodel.Task) error {
		result, runErr := c.buildPrecheckResult()
		if result == nil {
			result = &PrecheckResult{
				Ready:       false,
				GeneratedAt: time.Now(),
				Issues:      []string{"backup-service 预检执行失败"},
			}
		}
		if runErr != nil {
			result.Issues = append(result.Issues, runErr.Error())
		}
		task.DetailJSON = marshalJSON(result)

		switch {
		case runErr != nil:
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "环境预检执行失败"
			task.ErrorMessage = runErr.Error()
		case !result.Ready:
			task.Status = backupmodel.TaskStatusWarning
			task.Summary = fmt.Sprintf("环境预检完成，发现 %d 个阻塞项", len(result.Issues))
		default:
			task.Status = backupmodel.TaskStatusSucceeded
			task.Summary = fmt.Sprintf("环境预检完成，%d 个警告", len(result.Warnings))
		}
		return runErr
	})
	if err != nil {
		return nil, err
	}

	finishedAt := task.FinishedAt
	if finishedAt != nil {
		if _, stateErr := c.store.UpdateSystemState(ctx, func(state *backupmodel.SystemState) error {
			state.LastPrecheckAt = finishedAt
			state.LastPrecheckTaskID = &task.ID
			return nil
		}); stateErr != nil {
			return nil, stateErr
		}
	}

	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) CreateNamespaceSnapshot(ctx context.Context, requestedBy string, note string, relativePath string) (*TaskView, error) {
	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeNamespaceSnapshot, requestedBy, note, "namespace 快照执行中", func(task *backupmodel.Task) error {
		snapshot, snapshotErr := c.createNamespaceSnapshot(ctx, task, relativePath, backupmodel.SnapshotSourceManual)
		if snapshotErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "namespace 快照失败"
			task.ErrorMessage = snapshotErr.Error()
			return snapshotErr
		}

		result := NamespaceSnapshotResult{Snapshot: toSnapshotView(snapshot)}
		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("namespace 快照完成: %s", snapshot.RelativePath)
		task.DetailJSON = marshalJSON(result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) RestoreNamespaceSnapshot(ctx context.Context, requestedBy string, note string, snapshotID int64) (*TaskView, error) {
	state, err := c.store.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}
	if !state.MaintenanceMode {
		return nil, ErrMaintenanceModeRequired
	}

	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeNamespaceRestore, requestedBy, note, "namespace 恢复执行中", func(task *backupmodel.Task) error {
		restoreResult, restoreErr := c.restoreNamespaceSnapshot(ctx, task, snapshotID)
		if restoreErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "namespace 恢复失败"
			task.ErrorMessage = restoreErr.Error()
			return restoreErr
		}

		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("namespace 恢复完成: %s", restoreResult.RestoredPath)
		task.DetailJSON = marshalJSON(restoreResult)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) CreateMySQLSnapshot(ctx context.Context, requestedBy string, note string) (*TaskView, error) {
	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeMySQLSnapshot, requestedBy, note, "MySQL 快照执行中", func(task *backupmodel.Task) error {
		snapshot, databases, snapshotErr := c.createMySQLSnapshot(ctx, task, backupmodel.SnapshotSourceManual)
		if snapshotErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "MySQL 快照失败"
			task.ErrorMessage = snapshotErr.Error()
			return snapshotErr
		}

		result := MySQLSnapshotResult{
			Snapshot:  toSnapshotView(snapshot),
			Databases: databases,
		}
		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("MySQL 快照完成，库数量: %d", len(databases))
		task.DetailJSON = marshalJSON(result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) RestoreMySQLSnapshot(ctx context.Context, requestedBy string, note string, snapshotID int64) (*TaskView, error) {
	state, err := c.store.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}
	if !state.MaintenanceMode {
		return nil, ErrMaintenanceModeRequired
	}

	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeMySQLRestore, requestedBy, note, "MySQL 恢复执行中", func(task *backupmodel.Task) error {
		restoreResult, restoreErr := c.restoreMySQLSnapshot(ctx, task, snapshotID)
		if restoreErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "MySQL 恢复失败"
			task.ErrorMessage = restoreErr.Error()
			return restoreErr
		}

		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("MySQL 恢复完成，库数量: %d", len(restoreResult.Databases))
		task.DetailJSON = marshalJSON(restoreResult)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) CreateMinIOSnapshot(ctx context.Context, requestedBy string, note string) (*TaskView, error) {
	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeMinIOSnapshot, requestedBy, note, "MinIO 快照执行中", func(task *backupmodel.Task) error {
		snapshot, buckets, objectCount, snapshotErr := c.createMinIOSnapshot(ctx, task, backupmodel.SnapshotSourceManual)
		if snapshotErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "MinIO 快照失败"
			task.ErrorMessage = snapshotErr.Error()
			return snapshotErr
		}

		result := MinIOSnapshotResult{
			Snapshot: toSnapshotView(snapshot),
			Buckets:  buckets,
			Objects:  objectCount,
		}
		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("MinIO 快照完成，桶数量: %d，对象数量: %d", len(buckets), objectCount)
		task.DetailJSON = marshalJSON(result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) RestoreMinIOSnapshot(ctx context.Context, requestedBy string, note string, snapshotID int64) (*TaskView, error) {
	state, err := c.store.GetSystemState(ctx)
	if err != nil {
		return nil, err
	}
	if !state.MaintenanceMode {
		return nil, ErrMaintenanceModeRequired
	}

	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeMinIORestore, requestedBy, note, "MinIO 恢复执行中", func(task *backupmodel.Task) error {
		restoreResult, restoreErr := c.restoreMinIOSnapshot(ctx, task, snapshotID)
		if restoreErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "MinIO 恢复失败"
			task.ErrorMessage = restoreErr.Error()
			return restoreErr
		}

		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("MinIO 恢复完成，桶数量: %d，对象数量: %d", len(restoreResult.Buckets), restoreResult.Objects)
		task.DetailJSON = marshalJSON(restoreResult)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) DeleteSnapshot(ctx context.Context, requestedBy string, note string, snapshotID int64) (*TaskView, error) {
	task, err := c.runExclusiveTask(ctx, backupmodel.TaskTypeSnapshotDelete, requestedBy, note, "快照删除执行中", func(task *backupmodel.Task) error {
		deleteResult, deleteErr := c.deleteSnapshot(ctx, snapshotID)
		if deleteErr != nil {
			task.Status = backupmodel.TaskStatusFailed
			task.Summary = "快照删除失败"
			task.ErrorMessage = deleteErr.Error()
			return deleteErr
		}

		task.DetailJSON = marshalJSON(deleteResult)
		if deleteResult.ArchiveMissing {
			task.Status = backupmodel.TaskStatusWarning
			task.Summary = fmt.Sprintf("快照记录已删除，归档文件原本缺失: #%d", deleteResult.DeletedSnapshot.ID)
			return nil
		}

		task.Status = backupmodel.TaskStatusSucceeded
		task.Summary = fmt.Sprintf("快照已删除: #%d", deleteResult.DeletedSnapshot.ID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := toTaskView(task)
	return &view, nil
}

func (c *ControlPlane) runExclusiveTask(
	ctx context.Context,
	taskType string,
	requestedBy string,
	note string,
	initialSummary string,
	run func(task *backupmodel.Task) error,
) (*backupmodel.Task, error) {
	if !c.taskMu.TryLock() {
		return nil, ErrTaskAlreadyRunning
	}
	defer c.taskMu.Unlock()

	now := time.Now()
	task := &backupmodel.Task{
		Type:        taskType,
		Status:      backupmodel.TaskStatusRunning,
		RequestedBy: strings.TrimSpace(requestedBy),
		Note:        strings.TrimSpace(note),
		Summary:     initialSummary,
		StartedAt:   &now,
	}
	if task.RequestedBy == "" {
		task.RequestedBy = "backup-console"
	}

	if err := c.store.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	if _, err := c.store.UpdateSystemState(ctx, func(state *backupmodel.SystemState) error {
		state.ActiveTaskID = &task.ID
		return nil
	}); err != nil {
		return nil, err
	}

	runErr := run(task)

	finishedAt := time.Now()
	task.FinishedAt = &finishedAt
	if runErr != nil && task.ErrorMessage == "" {
		task.ErrorMessage = runErr.Error()
	}
	if task.Status == backupmodel.TaskStatusRunning {
		if runErr != nil {
			task.Status = backupmodel.TaskStatusFailed
		} else {
			task.Status = backupmodel.TaskStatusSucceeded
		}
	}

	saveErr := c.store.SaveTask(ctx, task)
	stateErr := c.clearActiveTask(ctx)

	if saveErr != nil {
		return nil, saveErr
	}
	if stateErr != nil {
		return nil, stateErr
	}
	if runErr != nil {
		return nil, runErr
	}

	return task, nil
}

func (c *ControlPlane) clearActiveTask(ctx context.Context) error {
	_, err := c.store.UpdateSystemState(ctx, func(state *backupmodel.SystemState) error {
		state.ActiveTaskID = nil
		return nil
	})
	return err
}

func (c *ControlPlane) createNamespaceSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	relativePath string,
	source string,
) (*backupmodel.Snapshot, error) {
	sanitized, err := sanitizeRelativePath(relativePath)
	if err != nil {
		return nil, err
	}

	sourceAbs, err := resolveWithinRoot(c.cfg.Storage.NamespacePath, sanitized)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(sourceAbs); err != nil {
		return nil, err
	}

	archiveDir := filepath.Join(c.cfg.Repository.RootPath, backupmodel.SnapshotResourceNamespace, time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, err
	}
	archiveName := fmt.Sprintf("%s-task-%d-%d.tar.gz", backupmodel.SnapshotResourceNamespace, task.ID, time.Now().Unix())
	archivePath := filepath.Join(archiveDir, archiveName)

	stats, err := createScopeArchive(c.cfg.Storage.NamespacePath, sanitized, archivePath)
	if err != nil {
		return nil, err
	}

	snapshot := &backupmodel.Snapshot{
		ResourceType:   backupmodel.SnapshotResourceNamespace,
		RelativePath:   sanitized,
		Source:         source,
		RequestedBy:    task.RequestedBy,
		Note:           task.Note,
		ArchivePath:    archivePath,
		ArchiveSize:    stats.ArchiveSize,
		FileCount:      stats.FileCount,
		DirectoryCount: stats.DirectoryCount,
	}
	if err := c.store.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (c *ControlPlane) restoreNamespaceSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	snapshotID int64,
) (*NamespaceRestoreResult, error) {
	snapshot, err := c.store.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	if snapshot.ResourceType != backupmodel.SnapshotResourceNamespace {
		return nil, fmt.Errorf("snapshot %d is not a namespace snapshot", snapshot.ID)
	}
	if _, err := os.Stat(snapshot.ArchivePath); err != nil {
		return nil, fmt.Errorf("snapshot archive not found: %w", err)
	}

	targetRelative := snapshot.RelativePath
	targetAbs, err := resolveWithinRoot(c.cfg.Storage.NamespacePath, targetRelative)
	if err != nil {
		return nil, err
	}

	var preRestoreSnapshot *backupmodel.Snapshot
	if _, err := os.Lstat(targetAbs); err == nil {
		preRestoreSnapshot, err = c.createNamespaceSnapshot(ctx, task, targetRelative, backupmodel.SnapshotSourcePreRestore)
		if err != nil {
			return nil, fmt.Errorf("create pre-restore snapshot: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	stagingDir := filepath.Join(c.cfg.Repository.StagingPath, "restore", fmt.Sprintf("task-%d-%d", task.ID, time.Now().Unix()))
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(stagingDir)

	if err := extractArchive(snapshot.ArchivePath, stagingDir); err != nil {
		return nil, err
	}

	stagedSource := stagingDir
	if targetRelative != "." {
		stagedSource = filepath.Join(stagingDir, filepath.FromSlash(targetRelative))
	}
	if _, err := os.Lstat(stagedSource); err != nil {
		return nil, fmt.Errorf("staged snapshot path missing: %w", err)
	}

	if targetRelative == "." {
		if err := removeDirectoryContents(c.cfg.Storage.NamespacePath); err != nil {
			return nil, err
		}
		if err := copyDirectoryContents(stagedSource, c.cfg.Storage.NamespacePath); err != nil {
			return nil, err
		}
	} else {
		if err := os.RemoveAll(targetAbs); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return nil, err
		}
		if err := copyPath(stagedSource, targetAbs); err != nil {
			return nil, err
		}
	}

	result := &NamespaceRestoreResult{
		RestoredSnapshot: toSnapshotView(snapshot),
		RestoredPath:     targetRelative,
	}
	if preRestoreSnapshot != nil {
		view := toSnapshotView(preRestoreSnapshot)
		result.PreRestoreSnapshot = &view
	}
	return result, nil
}

func (c *ControlPlane) createMySQLSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	source string,
) (*backupmodel.Snapshot, []string, error) {
	databases, err := c.collectMySQLDatabases(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(databases) == 0 {
		return nil, nil, fmt.Errorf("no non-system mysql databases found")
	}

	archiveDir := filepath.Join(c.cfg.Repository.RootPath, backupmodel.SnapshotResourceMySQL, time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, nil, err
	}
	archiveName := fmt.Sprintf("%s-task-%d-%d.sql.gz", backupmodel.SnapshotResourceMySQL, task.ID, time.Now().Unix())
	archivePath := filepath.Join(archiveDir, archiveName)

	if err := c.dumpMySQLDatabases(ctx, databases, archivePath); err != nil {
		return nil, nil, err
	}
	stat, err := os.Stat(archivePath)
	if err != nil {
		return nil, nil, err
	}

	snapshot := &backupmodel.Snapshot{
		ResourceType:   backupmodel.SnapshotResourceMySQL,
		RelativePath:   "logical-full",
		Source:         source,
		RequestedBy:    task.RequestedBy,
		Note:           task.Note,
		ArchivePath:    archivePath,
		ArchiveSize:    stat.Size(),
		FileCount:      1,
		DirectoryCount: 0,
		MetadataJSON: marshalJSON(map[string]interface{}{
			"databases": databases,
			"format":    "sql.gz",
		}),
	}
	if err := c.store.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, nil, err
	}
	return snapshot, databases, nil
}

func (c *ControlPlane) createMinIOSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	source string,
) (*backupmodel.Snapshot, []string, int64, error) {
	client, err := c.newMinIOClient()
	if err != nil {
		return nil, nil, 0, err
	}

	bucketInfos, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, nil, 0, err
	}

	archiveDir := filepath.Join(c.cfg.Repository.RootPath, backupmodel.SnapshotResourceMinIO, time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, nil, 0, err
	}
	archiveName := fmt.Sprintf("%s-task-%d-%d.tar.gz", backupmodel.SnapshotResourceMinIO, task.ID, time.Now().Unix())
	archivePath := filepath.Join(archiveDir, archiveName)

	manifest, stats, err := c.createMinIOArchive(ctx, client, bucketInfos, archivePath)
	if err != nil {
		return nil, nil, 0, err
	}

	buckets := make([]string, 0, len(manifest.Buckets))
	var objectCount int64
	for _, bucket := range manifest.Buckets {
		buckets = append(buckets, bucket.Name)
		objectCount += int64(len(bucket.Objects))
	}

	snapshot := &backupmodel.Snapshot{
		ResourceType:   backupmodel.SnapshotResourceMinIO,
		RelativePath:   "logical-full",
		Source:         source,
		RequestedBy:    task.RequestedBy,
		Note:           task.Note,
		ArchivePath:    archivePath,
		ArchiveSize:    stats.ArchiveSize,
		FileCount:      stats.FileCount,
		DirectoryCount: stats.DirectoryCount,
		MetadataJSON: marshalJSON(map[string]interface{}{
			"buckets": manifest.Buckets,
			"format":  "tar.gz",
			"objects": objectCount,
		}),
	}
	if err := c.store.CreateSnapshot(ctx, snapshot); err != nil {
		return nil, nil, 0, err
	}
	return snapshot, buckets, objectCount, nil
}

func (c *ControlPlane) restoreMySQLSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	snapshotID int64,
) (*MySQLRestoreResult, error) {
	snapshot, err := c.store.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	if snapshot.ResourceType != backupmodel.SnapshotResourceMySQL {
		return nil, fmt.Errorf("snapshot %d is not a mysql snapshot", snapshot.ID)
	}
	if _, err := os.Stat(snapshot.ArchivePath); err != nil {
		return nil, fmt.Errorf("snapshot archive not found: %w", err)
	}

	currentDatabases, err := c.collectMySQLDatabases(ctx)
	if err != nil {
		return nil, err
	}

	var preRestoreSnapshot *backupmodel.Snapshot
	if len(currentDatabases) > 0 {
		preRestoreSnapshot, _, err = c.createMySQLSnapshot(ctx, task, backupmodel.SnapshotSourcePreRestore)
		if err != nil {
			return nil, fmt.Errorf("create pre-restore mysql snapshot: %w", err)
		}
	}

	if err := c.dropMySQLDatabases(ctx, currentDatabases); err != nil {
		return nil, err
	}
	if err := c.restoreMySQLDump(ctx, snapshot.ArchivePath); err != nil {
		return nil, err
	}

	result := &MySQLRestoreResult{
		RestoredSnapshot: toSnapshotView(snapshot),
		Databases:        decodeSnapshotDatabases(snapshot),
	}
	if preRestoreSnapshot != nil {
		view := toSnapshotView(preRestoreSnapshot)
		result.PreRestoreSnapshot = &view
	}
	return result, nil
}

func (c *ControlPlane) restoreMinIOSnapshot(
	ctx context.Context,
	task *backupmodel.Task,
	snapshotID int64,
) (*MinIORestoreResult, error) {
	snapshot, err := c.store.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	if snapshot.ResourceType != backupmodel.SnapshotResourceMinIO {
		return nil, fmt.Errorf("snapshot %d is not a minio snapshot", snapshot.ID)
	}
	if _, err := os.Stat(snapshot.ArchivePath); err != nil {
		return nil, fmt.Errorf("snapshot archive not found: %w", err)
	}

	client, err := c.newMinIOClient()
	if err != nil {
		return nil, err
	}

	bucketInfos, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var preRestoreSnapshot *backupmodel.Snapshot
	if len(bucketInfos) > 0 {
		preRestoreSnapshot, _, _, err = c.createMinIOSnapshot(ctx, task, backupmodel.SnapshotSourcePreRestore)
		if err != nil {
			return nil, fmt.Errorf("create pre-restore minio snapshot: %w", err)
		}
	}

	if err := c.clearMinIOBuckets(ctx, client, bucketInfos); err != nil {
		return nil, err
	}

	stagingDir := filepath.Join(c.cfg.Repository.StagingPath, "minio-restore", fmt.Sprintf("task-%d-%d", task.ID, time.Now().Unix()))
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(stagingDir)

	if err := extractArchive(snapshot.ArchivePath, stagingDir); err != nil {
		return nil, err
	}

	manifest, err := loadMinIOManifest(filepath.Join(stagingDir, "_manifest.json"))
	if err != nil {
		return nil, err
	}

	var objectCount int64
	buckets := make([]string, 0, len(manifest.Buckets))
	for _, bucket := range manifest.Buckets {
		buckets = append(buckets, bucket.Name)

		exists, err := client.BucketExists(ctx, bucket.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			if err := client.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{}); err != nil {
				return nil, err
			}
		}

		for _, object := range bucket.Objects {
			objectPath, err := resolveWithinRoot(stagingDir, object.BlobPath)
			if err != nil {
				return nil, err
			}
			file, err := os.Open(objectPath)
			if err != nil {
				return nil, err
			}

			_, putErr := client.PutObject(ctx, bucket.Name, object.Key, file, object.Size, minio.PutObjectOptions{
				ContentType:  object.ContentType,
				UserMetadata: object.UserMetadata,
			})
			closeErr := file.Close()
			if putErr != nil {
				return nil, putErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			objectCount++
		}
	}

	result := &MinIORestoreResult{
		RestoredSnapshot: toSnapshotView(snapshot),
		Buckets:          buckets,
		Objects:          objectCount,
	}
	if preRestoreSnapshot != nil {
		view := toSnapshotView(preRestoreSnapshot)
		result.PreRestoreSnapshot = &view
	}
	return result, nil
}

func (c *ControlPlane) deleteSnapshot(ctx context.Context, snapshotID int64) (*SnapshotDeleteResult, error) {
	snapshot, err := c.store.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}

	result := &SnapshotDeleteResult{
		DeletedSnapshot: toSnapshotView(snapshot),
	}

	if archivePath := strings.TrimSpace(snapshot.ArchivePath); archivePath != "" {
		if err := os.Remove(archivePath); err != nil {
			if os.IsNotExist(err) {
				result.ArchiveMissing = true
			} else {
				return nil, fmt.Errorf("delete snapshot archive: %w", err)
			}
		} else {
			result.ArchiveDeleted = true
		}

		if err := cleanupEmptyParentDirs(filepath.Dir(archivePath), c.cfg.Repository.RootPath); err != nil {
			return nil, fmt.Errorf("cleanup snapshot archive directories: %w", err)
		}
	}

	if err := c.store.DeleteSnapshot(ctx, snapshot.ID); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ControlPlane) collectMySQLDatabases(ctx context.Context) ([]string, error) {
	args, env, err := c.mysqlConnectionArgs()
	if err != nil {
		return nil, err
	}
	args = append(args, "--batch", "--skip-column-names", "--execute", "SHOW DATABASES;")

	cmd := exec.CommandContext(ctx, c.cfg.GetMySQLBinary(), args...)
	cmd.Env = env

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list mysql databases: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	return filterMySQLDatabases(lines), nil
}

func filterMySQLDatabases(lines []string) []string {
	systemDBs := map[string]struct{}{
		"information_schema": {},
		"performance_schema": {},
		"mysql":              {},
		"sys":                {},
	}

	databases := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if _, isSystem := systemDBs[name]; isSystem {
			continue
		}
		databases = append(databases, name)
	}
	return databases
}

func (c *ControlPlane) dumpMySQLDatabases(ctx context.Context, databases []string, archivePath string) error {
	args, env, err := c.mysqlConnectionArgs()
	if err != nil {
		return err
	}
	args = append(args,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		"--hex-blob",
		"--set-gtid-purged=OFF",
		"--databases",
	)
	args = append(args, databases...)

	cmd := exec.CommandContext(ctx, c.cfg.GetMySQLDumpBinary(), args...)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)

	if err := cmd.Start(); err != nil {
		gz.Close()
		return err
	}

	if _, err := io.Copy(gz, stdout); err != nil {
		gz.Close()
		_ = cmd.Wait()
		return err
	}
	if err := gz.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("mysqldump failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (c *ControlPlane) newMinIOClient() (*minio.Client, error) {
	accessKey := c.cfg.GetMinIOAccessKey()
	secretKey := c.cfg.GetMinIOSecretKey()
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("minio credentials are required for minio backup operations")
	}

	client, err := minio.New(c.cfg.GetMinIOAddress(), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *ControlPlane) createMinIOArchive(
	ctx context.Context,
	client *minio.Client,
	bucketInfos []minio.BucketInfo,
	archivePath string,
) (*minIOSnapshotManifest, *snapshotArchiveStats, error) {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.Create(archivePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	manifest := &minIOSnapshotManifest{
		Buckets: make([]minIOBucketManifest, 0, len(bucketInfos)),
	}
	stats := &snapshotArchiveStats{}

	for _, bucketInfo := range bucketInfos {
		bucketManifest := minIOBucketManifest{
			Name:    bucketInfo.Name,
			Objects: make([]minIOObjectManifest, 0),
		}
		manifest.Buckets = append(manifest.Buckets, bucketManifest)
		stats.DirectoryCount++

		objectIndex := 0
		for objectInfo := range client.ListObjects(ctx, bucketInfo.Name, minio.ListObjectsOptions{Recursive: true}) {
			if objectInfo.Err != nil {
				return nil, nil, objectInfo.Err
			}

			blobPath := filepath.ToSlash(filepath.Join("objects", bucketInfo.Name, fmt.Sprintf("%06d.bin", objectIndex)))
			objectIndex++

			stat, err := client.StatObject(ctx, bucketInfo.Name, objectInfo.Key, minio.StatObjectOptions{})
			if err != nil {
				return nil, nil, err
			}

			object, err := client.GetObject(ctx, bucketInfo.Name, objectInfo.Key, minio.GetObjectOptions{})
			if err != nil {
				return nil, nil, err
			}

			header := &tar.Header{
				Name:    blobPath,
				Mode:    0o600,
				Size:    stat.Size,
				ModTime: stat.LastModified,
			}
			if err := tw.WriteHeader(header); err != nil {
				object.Close()
				return nil, nil, err
			}
			if _, err := io.Copy(tw, object); err != nil {
				object.Close()
				return nil, nil, err
			}
			if err := object.Close(); err != nil {
				return nil, nil, err
			}

			manifest.Buckets[len(manifest.Buckets)-1].Objects = append(manifest.Buckets[len(manifest.Buckets)-1].Objects, minIOObjectManifest{
				Key:          objectInfo.Key,
				BlobPath:     blobPath,
				Size:         stat.Size,
				ContentType:  stat.ContentType,
				UserMetadata: stat.UserMetadata,
			})
			stats.FileCount++
		}
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	manifestHeader := &tar.Header{
		Name:    "_manifest.json",
		Mode:    0o600,
		Size:    int64(len(manifestBytes)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(manifestHeader); err != nil {
		return nil, nil, err
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		return nil, nil, err
	}
	stats.FileCount++

	if err := tw.Close(); err != nil {
		return nil, nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, nil, err
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return nil, nil, err
	}
	stats.ArchiveSize = stat.Size()
	return manifest, stats, nil
}

func (c *ControlPlane) clearMinIOBuckets(ctx context.Context, client *minio.Client, bucketInfos []minio.BucketInfo) error {
	for _, bucketInfo := range bucketInfos {
		for objectInfo := range client.ListObjects(ctx, bucketInfo.Name, minio.ListObjectsOptions{Recursive: true}) {
			if objectInfo.Err != nil {
				return objectInfo.Err
			}
			if err := client.RemoveObject(ctx, bucketInfo.Name, objectInfo.Key, minio.RemoveObjectOptions{}); err != nil {
				return err
			}
		}
		if err := client.RemoveBucket(ctx, bucketInfo.Name); err != nil {
			return err
		}
	}
	return nil
}

func loadMinIOManifest(path string) (*minIOSnapshotManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest minIOSnapshotManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (c *ControlPlane) dropMySQLDatabases(ctx context.Context, databases []string) error {
	for _, database := range databases {
		statement := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", escapeMySQLIdentifier(database))
		if err := c.execMySQLStatement(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (c *ControlPlane) restoreMySQLDump(ctx context.Context, archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	args, env, err := c.mysqlConnectionArgs()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, c.cfg.GetMySQLBinary(), args...)
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(stdin, gz); err != nil {
		stdin.Close()
		_ = cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("mysql restore failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (c *ControlPlane) execMySQLStatement(ctx context.Context, statement string) error {
	args, env, err := c.mysqlConnectionArgs()
	if err != nil {
		return err
	}
	args = append(args, "--batch", "--silent", "--execute", statement)

	cmd := exec.CommandContext(ctx, c.cfg.GetMySQLBinary(), args...)
	cmd.Env = env

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql execute failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (c *ControlPlane) mysqlConnectionArgs() ([]string, []string, error) {
	password := c.cfg.GetMySQLPassword()
	if password == "" {
		return nil, nil, fmt.Errorf("mysql password is required for mysql backup operations")
	}

	host, port, err := net.SplitHostPort(c.cfg.GetMySQLAddress())
	if err != nil {
		return nil, nil, err
	}

	args := []string{
		"--host", host,
		"--port", port,
		"--user", c.cfg.GetMySQLUser(),
	}
	env := append(os.Environ(), "MYSQL_PWD="+password)
	return args, env, nil
}

func escapeMySQLIdentifier(name string) string {
	return strings.ReplaceAll(name, "`", "``")
}

func decodeSnapshotDatabases(snapshot *backupmodel.Snapshot) []string {
	if strings.TrimSpace(snapshot.MetadataJSON) == "" {
		return nil
	}

	var raw struct {
		Databases []string `json:"databases"`
	}
	if err := json.Unmarshal([]byte(snapshot.MetadataJSON), &raw); err != nil {
		return nil
	}
	return raw.Databases
}

func (c *ControlPlane) buildPrecheckResult() (*PrecheckResult, error) {
	result := &PrecheckResult{
		Ready:        true,
		GeneratedAt:  time.Now(),
		Paths:        make(map[string]PathCheck),
		Dependencies: make(map[string]DependencyCheck),
		Tooling:      make(map[string]ToolCheck),
		Filesystems:  make(map[string]FilesystemCheck),
	}

	pathChecks := []struct {
		name         string
		path         string
		wantWritable bool
	}{
		{name: "namespace", path: c.cfg.Storage.NamespacePath, wantWritable: false},
		{name: "data", path: c.cfg.Storage.DataPath, wantWritable: true},
		{name: "logs", path: c.cfg.Storage.LogsPath, wantWritable: true},
		{name: "mysql", path: c.cfg.Storage.MySQLPath, wantWritable: false},
		{name: "minio", path: c.cfg.Storage.MinIOPath, wantWritable: false},
		{name: "podman_storage", path: c.cfg.Storage.PodmanStoragePath, wantWritable: false},
		{name: "repo", path: c.cfg.Repository.RootPath, wantWritable: true},
		{name: "state", path: c.cfg.Repository.StatePath, wantWritable: true},
		{name: "staging", path: c.cfg.Repository.StagingPath, wantWritable: true},
	}
	for _, item := range pathChecks {
		check := inspectPath(item.path, item.wantWritable)
		result.Paths[item.name] = check
		if !check.Exists || !check.IsDir {
			result.Ready = false
			result.Issues = append(result.Issues, fmt.Sprintf("%s 路径不可用: %s", item.name, item.path))
			continue
		}
		if item.wantWritable && !check.Writable {
			result.Ready = false
			result.Issues = append(result.Issues, fmt.Sprintf("%s 路径不可写: %s", item.name, item.path))
		}
	}

	filesystemTargets := []struct {
		name string
		path string
	}{
		{name: "data", path: c.cfg.Storage.DataPath},
		{name: "mysql", path: c.cfg.Storage.MySQLPath},
		{name: "minio", path: c.cfg.Storage.MinIOPath},
	}
	for _, item := range filesystemTargets {
		fsCheck := inspectFilesystem(item.path)
		result.Filesystems[item.name] = fsCheck
		if fsCheck.Error != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s 文件系统容量检查失败: %s", item.name, fsCheck.Error))
			continue
		}
		if fsCheck.TotalBytes > 0 {
			freeRatio := float64(fsCheck.AvailableBytes) / float64(fsCheck.TotalBytes)
			if freeRatio < 0.10 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s 剩余空间低于 10%%", item.name))
			}
		}
	}

	dependencies := map[string]string{
		"mysql": c.cfg.GetMySQLAddress(),
		"minio": c.cfg.GetMinIOAddress(),
	}
	for name, address := range dependencies {
		check := inspectDependency(address)
		result.Dependencies[name] = check
		if !check.Reachable {
			result.Ready = false
			result.Issues = append(result.Issues, fmt.Sprintf("%s 不可达: %s", name, address))
		}
	}

	tools := map[string]string{
		"mysql":     c.cfg.GetMySQLBinary(),
		"mysqldump": c.cfg.GetMySQLDumpBinary(),
		"restic":    c.cfg.GetResticBinary(),
		"mc":        c.cfg.GetMinIOClientBinary(),
	}
	for name, binary := range tools {
		check := inspectBinary(binary)
		result.Tooling[name] = check
		if !check.Exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s 未安装，相关备份能力暂不可用", name))
		}
	}

	return result, nil
}

func (c *ControlPlane) statusPathReport() map[string]PathCheck {
	return map[string]PathCheck{
		"namespace":      inspectPath(c.cfg.Storage.NamespacePath, false),
		"data":           inspectPath(c.cfg.Storage.DataPath, true),
		"logs":           inspectPath(c.cfg.Storage.LogsPath, true),
		"mysql":          inspectPath(c.cfg.Storage.MySQLPath, false),
		"minio":          inspectPath(c.cfg.Storage.MinIOPath, false),
		"podman_storage": inspectPath(c.cfg.Storage.PodmanStoragePath, false),
		"repo":           inspectPath(c.cfg.Repository.RootPath, true),
		"state":          inspectPath(c.cfg.Repository.StatePath, true),
		"staging":        inspectPath(c.cfg.Repository.StagingPath, true),
	}
}

func (c *ControlPlane) syncMaintenanceArtifacts(state *backupmodel.SystemState) error {
	markerPath := c.cfg.GetMaintenanceMarkerPath()
	pagePath := c.cfg.GetMaintenancePagePath()
	metadataPath := c.cfg.GetMaintenanceMetadataPath()

	if !state.MaintenanceMode {
		if err := removeIfExists(markerPath); err != nil {
			return err
		}
		if err := removeIfExists(pagePath); err != nil {
			return err
		}
		if err := removeIfExists(metadataPath); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pagePath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return err
	}

	payload := maintenanceStateFile{
		Enabled:   true,
		Reason:    strings.TrimSpace(state.MaintenanceReason),
		UpdatedAt: state.MaintenanceUpdatedAt,
	}
	if err := writeAtomicFile(metadataPath, []byte(marshalJSON(payload)+"\n"), 0o644); err != nil {
		return err
	}
	if err := writeAtomicFile(pagePath, []byte(renderMaintenanceHTML(payload)), 0o644); err != nil {
		return err
	}
	return writeAtomicFile(markerPath, []byte("enabled\n"), 0o644)
}

func renderMaintenanceHTML(state maintenanceStateFile) string {
	reason := "系统正在执行备份恢复操作，请稍后再试。"
	if trimmed := strings.TrimSpace(state.Reason); trimmed != "" {
		reason = trimmed
	}

	updatedAt := "-"
	if state.UpdatedAt != nil {
		updatedAt = state.UpdatedAt.Local().Format("2006-01-02 15:04:05")
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>系统维护中</title>
  <style>
    :root {
      --bg: #f4efe6;
      --panel: rgba(255, 252, 246, 0.92);
      --line: rgba(46, 39, 28, 0.12);
      --text: #2e241b;
      --muted: #756656;
      --accent: #9d2e2e;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
      color: var(--text);
      font-family: "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(157, 46, 46, 0.12), transparent 30%%),
        linear-gradient(180deg, #f8f4ee 0%%, #efe7da 100%%);
    }
    .panel {
      width: min(720px, 100%%);
      padding: 36px 32px;
      border-radius: 24px;
      background: var(--panel);
      border: 1px solid var(--line);
      box-shadow: 0 24px 56px rgba(58, 42, 24, 0.14);
    }
    h1 {
      margin: 0;
      font-size: clamp(30px, 5vw, 44px);
      letter-spacing: -0.04em;
    }
    p {
      margin: 14px 0 0;
      line-height: 1.7;
      color: var(--muted);
      font-size: 16px;
    }
    .badge {
      display: inline-flex;
      padding: 8px 14px;
      border-radius: 999px;
      font-size: 13px;
      font-weight: 700;
      color: var(--accent);
      background: rgba(157, 46, 46, 0.12);
      margin-bottom: 18px;
    }
    .meta {
      margin-top: 24px;
      font-size: 14px;
      color: var(--muted);
    }
  </style>
</head>
<body>
  <main class="panel">
    <div class="badge">AI Agent OS</div>
    <h1>系统维护中</h1>
    <p>%s</p>
    <p class="meta">维护开始时间：%s</p>
  </main>
</body>
</html>
`, html.EscapeString(reason), html.EscapeString(updatedAt))
}

func createScopeArchive(rootPath string, relativePath string, archivePath string) (*snapshotArchiveStats, error) {
	sourceAbs, err := resolveWithinRoot(rootPath, relativePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(sourceAbs)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return nil, err
	}

	f, err := os.Create(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	stats := &snapshotArchiveStats{}
	headerBase := relativePath
	if headerBase == "." {
		headerBase = ""
	}

	writeOne := func(pathAbs string, fileInfo fs.FileInfo, headerName string) error {
		linkTarget := ""
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(pathAbs)
			if err != nil {
				return err
			}
			linkTarget = target
		}

		header, err := tar.FileInfoHeader(fileInfo, linkTarget)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(headerName)
		if fileInfo.IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		switch {
		case fileInfo.IsDir():
			stats.DirectoryCount++
			return nil
		case fileInfo.Mode()&os.ModeSymlink != 0:
			stats.FileCount++
			return nil
		case fileInfo.Mode().IsRegular():
			stats.FileCount++
			file, err := os.Open(pathAbs)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tw, file)
			return err
		default:
			return nil
		}
	}

	if info.IsDir() {
		err = filepath.Walk(sourceAbs, func(pathAbs string, fileInfo fs.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			subRel, err := filepath.Rel(sourceAbs, pathAbs)
			if err != nil {
				return err
			}

			headerName := headerBase
			if subRel != "." {
				headerName = filepath.Join(headerBase, subRel)
			}
			if headerName == "" {
				return nil
			}
			return writeOne(pathAbs, fileInfo, headerName)
		})
	} else {
		err = writeOne(sourceAbs, info, relativePath)
	}
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return nil, err
	}
	stats.ArchiveSize = stat.Size()
	return stats, nil
}

func extractArchive(archivePath string, destination string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		relativeName, err := sanitizeRelativePath(header.Name)
		if err != nil {
			return err
		}
		targetPath, err := resolveWithinRoot(destination, relativeName)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDirectoryContents(srcRoot string, dstRoot string) error {
	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcRoot, entry.Name())
		dstPath := filepath.Join(dstRoot, entry.Name())
		if err := copyPath(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(src string, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	switch {
	case info.IsDir():
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.Mode().IsRegular():
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	default:
		return nil
	}
}

func removeDirectoryContents(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func cleanupEmptyParentDirs(startDir string, stopDir string) error {
	if strings.TrimSpace(startDir) == "" || strings.TrimSpace(stopDir) == "" {
		return nil
	}

	current, err := filepath.Abs(startDir)
	if err != nil {
		return err
	}
	stop, err := filepath.Abs(stopDir)
	if err != nil {
		return err
	}

	for current != "." && current != string(os.PathSeparator) {
		if current == stop {
			return nil
		}

		err := os.Remove(current)
		if err != nil {
			if os.IsNotExist(err) {
				current = filepath.Dir(current)
				continue
			}
			if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
				return nil
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.ENOTEMPTY) {
				return nil
			}
			return err
		}
		current = filepath.Dir(current)
	}

	return nil
}

func sanitizeRelativePath(input string) (string, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return ".", nil
	}

	clean := path.Clean(trimmed)
	if clean == "." || clean == "/" {
		return ".", nil
	}
	relative := strings.TrimPrefix(clean, "/")
	if relative == "" || relative == "." {
		return ".", nil
	}
	if relative == ".." || strings.HasPrefix(relative, "../") {
		return "", ErrInvalidRelativePath
	}
	return relative, nil
}

func resolveWithinRoot(root string, relative string) (string, error) {
	sanitized, err := sanitizeRelativePath(relative)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := rootAbs
	if sanitized != "." {
		target = filepath.Join(rootAbs, filepath.FromSlash(sanitized))
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(os.PathSeparator)) {
		return "", ErrInvalidRelativePath
	}
	return targetAbs, nil
}

func toSystemStateView(state *backupmodel.SystemState) SystemStateView {
	return SystemStateView{
		MaintenanceMode:      state.MaintenanceMode,
		MaintenanceReason:    state.MaintenanceReason,
		ActiveTaskID:         state.ActiveTaskID,
		LastPrecheckAt:       state.LastPrecheckAt,
		LastPrecheckTaskID:   state.LastPrecheckTaskID,
		MaintenanceUpdatedAt: state.MaintenanceUpdatedAt,
		UpdatedAt:            state.UpdatedAt,
	}
}

func toTaskViews(tasks []backupmodel.Task) []TaskView {
	views := make([]TaskView, 0, len(tasks))
	for _, task := range tasks {
		views = append(views, toTaskView(&task))
	}
	return views
}

func toTaskView(task *backupmodel.Task) TaskView {
	return TaskView{
		ID:           task.ID,
		Type:         task.Type,
		Status:       task.Status,
		RequestedBy:  task.RequestedBy,
		Note:         task.Note,
		Summary:      task.Summary,
		ErrorMessage: task.ErrorMessage,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
		CreatedAt:    time.Time(task.CreatedAt),
		UpdatedAt:    time.Time(task.UpdatedAt),
		Detail:       decodeTaskDetail(task.DetailJSON),
	}
}

func toSnapshotViews(snapshots []backupmodel.Snapshot) []SnapshotView {
	views := make([]SnapshotView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		views = append(views, toSnapshotView(&snapshot))
	}
	return views
}

func toSnapshotView(snapshot *backupmodel.Snapshot) SnapshotView {
	return SnapshotView{
		ID:             snapshot.ID,
		ResourceType:   snapshot.ResourceType,
		RelativePath:   snapshot.RelativePath,
		Source:         snapshot.Source,
		RequestedBy:    snapshot.RequestedBy,
		Note:           snapshot.Note,
		ArchivePath:    snapshot.ArchivePath,
		ArchiveSize:    snapshot.ArchiveSize,
		FileCount:      snapshot.FileCount,
		DirectoryCount: snapshot.DirectoryCount,
		Metadata:       decodeSnapshotMetadata(snapshot.MetadataJSON),
		CreatedAt:      time.Time(snapshot.CreatedAt),
		UpdatedAt:      time.Time(snapshot.UpdatedAt),
	}
}

func decodeSnapshotMetadata(detail string) interface{} {
	if strings.TrimSpace(detail) == "" {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(detail), &decoded); err != nil {
		return detail
	}
	return decoded
}

func decodeTaskDetail(detail string) interface{} {
	if strings.TrimSpace(detail) == "" {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(detail), &decoded); err != nil {
		return detail
	}
	return decoded
}

func marshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeAtomicFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".backup-service-*")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func inspectPath(path string, wantWritable bool) PathCheck {
	check := PathCheck{
		Path:             path,
		WritableExpected: wantWritable,
	}

	info, err := os.Stat(path)
	if err != nil {
		check.Error = err.Error()
		return check
	}

	check.Exists = true
	check.IsDir = info.IsDir()
	if !info.IsDir() || !wantWritable {
		return check
	}

	f, err := os.CreateTemp(path, ".backup-service-writecheck-*")
	if err != nil {
		check.Error = err.Error()
		return check
	}
	check.Writable = true
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return check
}

func inspectFilesystem(path string) FilesystemCheck {
	check := FilesystemCheck{Path: path}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		check.Error = err.Error()
		return check
	}
	check.TotalBytes = fs.Blocks * uint64(fs.Bsize)
	check.FreeBytes = fs.Bfree * uint64(fs.Bsize)
	check.AvailableBytes = fs.Bavail * uint64(fs.Bsize)
	return check
}

func inspectDependency(address string) DependencyCheck {
	check := DependencyCheck{Address: address}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		check.Error = err.Error()
		return check
	}
	check.Reachable = true
	check.LatencyMS = time.Since(start).Milliseconds()
	_ = conn.Close()
	return check
}

func inspectBinary(binary string) ToolCheck {
	check := ToolCheck{Binary: binary}
	path, err := exec.LookPath(binary)
	if err != nil {
		check.Error = err.Error()
		return check
	}
	check.Exists = true
	check.Path = path
	return check
}
