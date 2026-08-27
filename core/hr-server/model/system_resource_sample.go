package model

import "time"

// SystemResourceSample stores the compact ten-minute runtime rollup.
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
	CPUUsedPercent    float64   `gorm:"column:cpu_used_percent;not null"`
	CPUMaxPercent     float64   `gorm:"column:cpu_max_percent;not null"`
	NetworkRxBytesPS  float64   `gorm:"column:network_rx_bytes_per_second;not null"`
	NetworkTxBytesPS  float64   `gorm:"column:network_tx_bytes_per_second;not null"`
	DiskReadBytesPS   float64   `gorm:"column:disk_read_bytes_per_second;not null"`
	DiskWriteBytesPS  float64   `gorm:"column:disk_write_bytes_per_second;not null"`
	Load1             float64   `gorm:"column:load_1;not null"`
}

func (SystemResourceSample) TableName() string {
	return "system_resource_samples"
}

type SystemCapacitySnapshot struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	CollectedAt time.Time `gorm:"column:collected_at;index;not null"`
	PayloadJSON string    `gorm:"column:payload_json;type:longtext;not null"`
}

func (SystemCapacitySnapshot) TableName() string { return "system_capacity_snapshots" }

type SystemPlatformSnapshot struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	CollectedAt time.Time `gorm:"column:collected_at;index;not null"`
	PayloadJSON string    `gorm:"column:payload_json;type:longtext;not null"`
}

func (SystemPlatformSnapshot) TableName() string { return "system_platform_snapshots" }
