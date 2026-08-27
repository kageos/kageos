package model

import "time"

// SystemResourceSample stores a compact, long-lived operational history.
// ComponentsJSON is a snapshot of the storage breakdown so the schema can grow
// without a migration for each newly monitored storage category.
type SystemResourceSample struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement"`
	CollectedAt       time.Time `gorm:"column:collected_at;index;not null"`
	DiskTotalBytes    uint64    `gorm:"column:disk_total_bytes;not null"`
	DiskUsedBytes     uint64    `gorm:"column:disk_used_bytes;not null"`
	DiskFreeBytes     uint64    `gorm:"column:disk_free_bytes;not null"`
	DiskUsedPercent   float64   `gorm:"column:disk_used_percent;not null"`
	MemoryTotalBytes  uint64    `gorm:"column:memory_total_bytes;not null"`
	MemoryUsedBytes   uint64    `gorm:"column:memory_used_bytes;not null"`
	MemoryUsedPercent float64   `gorm:"column:memory_used_percent;not null"`
	Load1             float64   `gorm:"column:load_1;not null"`
	ComponentsJSON    string    `gorm:"column:components_json;type:longtext"`
}

func (SystemResourceSample) TableName() string {
	return "system_resource_samples"
}
