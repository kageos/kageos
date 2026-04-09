package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

const (
	DefaultSystemStateID int64 = 1

	TaskTypePrecheck          = "precheck"
	TaskTypeNamespaceSnapshot = "namespace_snapshot"
	TaskTypeNamespaceRestore  = "namespace_restore"
	TaskTypeMySQLSnapshot     = "mysql_snapshot"
	TaskTypeMySQLRestore      = "mysql_restore"
	TaskTypeMinIOSnapshot     = "minio_snapshot"
	TaskTypeMinIORestore      = "minio_restore"
	TaskTypeSnapshotDelete    = "snapshot_delete"
	TaskTypeSnapshotPrune     = "snapshot_prune"

	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusWarning   = "warning"
	TaskStatusFailed    = "failed"

	SnapshotResourceNamespace = "namespace"
	SnapshotResourceMySQL     = "mysql"
	SnapshotResourceMinIO     = "minio"
	SnapshotSourceManual      = "manual"
	SnapshotSourcePreRestore  = "pre_restore"
)

type Task struct {
	models.Base
	Type         string     `gorm:"size:32;not null;index" json:"type"`
	Status       string     `gorm:"size:32;not null;index" json:"status"`
	RequestedBy  string     `gorm:"size:128" json:"requested_by"`
	Note         string     `gorm:"type:text" json:"note"`
	Summary      string     `gorm:"size:255" json:"summary"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	DetailJSON   string     `gorm:"type:text" json:"-"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

func (Task) TableName() string {
	return "backup_tasks"
}

type SystemState struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MaintenanceMode      bool       `gorm:"not null;default:false" json:"maintenance_mode"`
	MaintenanceReason    string     `gorm:"type:text" json:"maintenance_reason"`
	ActiveTaskID         *int64     `json:"active_task_id"`
	LastPrecheckAt       *time.Time `json:"last_precheck_at"`
	LastPrecheckTaskID   *int64     `json:"last_precheck_task_id"`
	MaintenanceUpdatedAt *time.Time `json:"maintenance_updated_at"`
	CreatedAt            time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SystemState) TableName() string {
	return "backup_system_state"
}

type Snapshot struct {
	models.Base
	ResourceType   string `gorm:"size:32;not null;index" json:"resource_type"`
	RelativePath   string `gorm:"size:512;not null;index" json:"relative_path"`
	Source         string `gorm:"size:32;not null;index" json:"source"`
	RequestedBy    string `gorm:"size:128" json:"requested_by"`
	Note           string `gorm:"type:text" json:"note"`
	ArchivePath    string `gorm:"size:1024;not null" json:"archive_path"`
	ArchiveSize    int64  `json:"archive_size"`
	FileCount      int64  `json:"file_count"`
	DirectoryCount int64  `json:"directory_count"`
	MetadataJSON   string `gorm:"type:text" json:"-"`
}

func (Snapshot) TableName() string {
	return "backup_snapshots"
}
