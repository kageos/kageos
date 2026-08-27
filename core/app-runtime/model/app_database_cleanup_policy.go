package model

import "github.com/kageos/kageos/pkg/gormx/models"

// AppDatabaseCleanupPolicy stores an explicit table-level recycle-bin policy.
// Tables without an override continue to use the runtime deployment defaults.
type AppDatabaseCleanupPolicy struct {
	models.Base
	AppDatabaseID int64  `gorm:"not null;uniqueIndex:idx_app_db_cleanup_policy" json:"app_database_id"`
	TargetTable   string `gorm:"column:table_name;size:64;not null;uniqueIndex:idx_app_db_cleanup_policy" json:"table_name"`
	Enabled       bool   `gorm:"not null;default:false" json:"enabled"`
	Mode          string `gorm:"size:16;not null;default:'dry_run'" json:"mode"`
	RetentionDays int    `gorm:"not null;default:30" json:"retention_days"`
	UpdatedBy     string `gorm:"size:100" json:"updated_by"`
}

func (AppDatabaseCleanupPolicy) TableName() string {
	return "app_database_cleanup_policies"
}
