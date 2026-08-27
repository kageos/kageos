package dto

import "time"

type EmailSettings struct {
	Mode        string `json:"mode"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	PasswordSet bool   `json:"password_set"`
	From        string `json:"from"`
	FromName    string `json:"from_name"`
}

type SystemSettingsResp struct {
	RegistrationMode string        `json:"registration_mode"`
	Email            EmailSettings `json:"email"`
}

type LoginAnnouncement struct {
	Enabled  bool   `json:"enabled"`
	Markdown string `json:"markdown"`
}

type UpdateLoginAnnouncementReq struct {
	Enabled  bool   `json:"enabled"`
	Markdown string `json:"markdown"`
}

type TLSCertificateInfo struct {
	Subject      string   `json:"subject"`
	Issuer       string   `json:"issuer"`
	DNSNames     []string `json:"dns_names"`
	IPAddresses  []string `json:"ip_addresses"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	IsSelfSigned bool     `json:"is_self_signed"`
}

type TLSSettingsResp struct {
	Mode            string              `json:"mode"`
	BaseURL         string              `json:"base_url"`
	CertFile        string              `json:"cert_file"`
	KeyFile         string              `json:"key_file"`
	CertExists      bool                `json:"cert_exists"`
	KeyExists       bool                `json:"key_exists"`
	Ready           bool                `json:"ready"`
	Writable        bool                `json:"writable"`
	ReloadSupported bool                `json:"reload_supported"`
	Certificate     *TLSCertificateInfo `json:"certificate,omitempty"`
	Message         string              `json:"message,omitempty"`
}

type UpdateTLSCertificateReq struct {
	CertificatePEM string `json:"certificate_pem" binding:"required"`
	PrivateKeyPEM  string `json:"private_key_pem" binding:"required"`
	Reload         bool   `json:"reload"`
}

type UpdateSystemSettingsReq struct {
	RegistrationMode string        `json:"registration_mode" binding:"required,oneof=admin_only email_code debug_code"`
	Email            EmailSettings `json:"email"`
}

type TestEmailReq struct {
	To string `json:"to" binding:"required,email"`
}

type SystemResourceComponent struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	PoolKey   string `json:"pool_key"`
	UsedBytes uint64 `json:"used_bytes"`
	Available bool   `json:"available"`
}

type SystemEnvironmentInfo struct {
	Mode              string `json:"mode"`
	Deployment        string `json:"deployment"`
	Containerized     bool   `json:"containerized"`
	ContainerEngine   string `json:"container_engine"`
	ContainerRemote   bool   `json:"container_remote"`
	StorageRootSource string `json:"storage_root_source"`
}

type SystemStoragePool struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Path        string  `json:"path,omitempty"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
	Primary     bool    `json:"primary"`
	Available   bool    `json:"available"`
}

type SystemResourceSnapshot struct {
	CollectedAt           time.Time                 `json:"collected_at"`
	Hostname              string                    `json:"hostname"`
	OperatingSystem       string                    `json:"operating_system"`
	Architecture          string                    `json:"architecture"`
	CPUCores              int                       `json:"cpu_cores"`
	CPUUsedPercent        float64                   `json:"cpu_used_percent"`
	CPUAvailable          bool                      `json:"cpu_available"`
	Load1                 float64                   `json:"load_1"`
	Load5                 float64                   `json:"load_5"`
	Load15                float64                   `json:"load_15"`
	LoadAvailable         bool                      `json:"load_available"`
	UptimeSeconds         uint64                    `json:"uptime_seconds"`
	MemoryTotalBytes      uint64                    `json:"memory_total_bytes"`
	MemoryUsedBytes       uint64                    `json:"memory_used_bytes"`
	MemoryUsedPercent     float64                   `json:"memory_used_percent"`
	MemoryAvailable       bool                      `json:"memory_available"`
	SwapTotalBytes        uint64                    `json:"swap_total_bytes"`
	SwapUsedBytes         uint64                    `json:"swap_used_bytes"`
	SwapUsedPercent       float64                   `json:"swap_used_percent"`
	NetworkRxBytes        uint64                    `json:"network_rx_bytes"`
	NetworkTxBytes        uint64                    `json:"network_tx_bytes"`
	NetworkAvailable      bool                      `json:"network_available"`
	NetworkRxBytesPS      float64                   `json:"network_rx_bytes_per_second"`
	NetworkTxBytesPS      float64                   `json:"network_tx_bytes_per_second"`
	DiskReadBytes         uint64                    `json:"disk_read_bytes"`
	DiskWriteBytes        uint64                    `json:"disk_write_bytes"`
	DiskIOAvailable       bool                      `json:"disk_io_available"`
	DiskReadBytesPS       float64                   `json:"disk_read_bytes_per_second"`
	DiskWriteBytesPS      float64                   `json:"disk_write_bytes_per_second"`
	DiskMount             string                    `json:"disk_mount"`
	DiskTotalBytes        uint64                    `json:"disk_total_bytes"`
	DiskUsedBytes         uint64                    `json:"disk_used_bytes"`
	DiskFreeBytes         uint64                    `json:"disk_free_bytes"`
	DiskUsedPercent       float64                   `json:"disk_used_percent"`
	Environment           SystemEnvironmentInfo     `json:"environment"`
	StoragePools          []SystemStoragePool       `json:"storage_pools"`
	Components            []SystemResourceComponent `json:"components"`
	DatabaseLogicalBytes  uint64                    `json:"database_logical_bytes"`
	DatabaseSizeAvailable bool                      `json:"database_size_available"`
	LargestDatabases      []SystemDatabaseSize      `json:"largest_databases"`
}

type SystemResourceHistoryPoint struct {
	CollectedAt       time.Time `json:"collected_at"`
	DiskUsedBytes     uint64    `json:"disk_used_bytes"`
	DiskUsedPercent   float64   `json:"disk_used_percent"`
	MemoryUsedPercent float64   `json:"memory_used_percent"`
	CPUUsedPercent    float64   `json:"cpu_used_percent"`
	CPUMaxPercent     float64   `json:"cpu_max_percent"`
	NetworkRxBytesPS  float64   `json:"network_rx_bytes_per_second"`
	NetworkTxBytesPS  float64   `json:"network_tx_bytes_per_second"`
	DiskReadBytesPS   float64   `json:"disk_read_bytes_per_second"`
	DiskWriteBytesPS  float64   `json:"disk_write_bytes_per_second"`
	Load1             float64   `json:"load_1"`
}

type SystemDatabaseSize struct {
	Name      string `json:"name"`
	UsedBytes uint64 `json:"used_bytes"`
}

type SystemPlatformServiceStats struct {
	WorkspacesTotal      int64 `json:"workspaces_total"`
	WorkspacesEnabled    int64 `json:"workspaces_enabled"`
	ServiceDirectories   int64 `json:"service_directories"`
	FunctionsTotal       int64 `json:"functions_total"`
	AppDatabasesTotal    int64 `json:"app_databases_total"`
	ScheduledTasksTotal  int64 `json:"scheduled_tasks_total"`
	ScheduledTasksActive int64 `json:"scheduled_tasks_active"`
}

type SystemDatabaseCapacityStats struct {
	Available  bool                 `json:"available"`
	TotalBytes uint64               `json:"total_bytes"`
	Databases  []SystemDatabaseSize `json:"databases"`
}

type SystemPlatformMetrics struct {
	CollectedAt           time.Time `json:"collected_at"`
	UsersTotal            int64     `json:"users_total"`
	UsersActive           int64     `json:"users_active"`
	UsersPending          int64     `json:"users_pending"`
	WorkspacesTotal       int64     `json:"workspaces_total"`
	WorkspacesEnabled     int64     `json:"workspaces_enabled"`
	ServiceDirectories    int64     `json:"service_directories"`
	FunctionsTotal        int64     `json:"functions_total"`
	AppDatabasesTotal     int64     `json:"app_databases_total"`
	ScheduledTasksTotal   int64     `json:"scheduled_tasks_total"`
	ScheduledTasksActive  int64     `json:"scheduled_tasks_active"`
	AppStatsAvailable     bool      `json:"app_stats_available"`
	RuntimeStatsAvailable bool      `json:"runtime_stats_available"`
	TimerStatsAvailable   bool      `json:"timer_stats_available"`
}

type SystemCollectionTaskStatus struct {
	Key             string     `json:"key"`
	Status          string     `json:"status"`
	LastStartedAt   *time.Time `json:"last_started_at,omitempty"`
	LastSucceededAt *time.Time `json:"last_succeeded_at,omitempty"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	DurationMillis  int64      `json:"duration_millis"`
	Error           string     `json:"error,omitempty"`
}

type StorageExpansionForecast struct {
	Status             string  `json:"status"`
	PoolKey            string  `json:"pool_key"`
	CurrentUsedPercent float64 `json:"current_used_percent"`
	DailyGrowthByte    float64 `json:"daily_growth_bytes"`
	TargetPercent      float64 `json:"target_percent"`
	DaysToTarget       *int    `json:"days_to_target,omitempty"`
	Message            string  `json:"message"`
}

type SystemResourceOverviewResp struct {
	Current                SystemResourceSnapshot       `json:"current"`
	History                []SystemResourceHistoryPoint `json:"history"`
	HistoryHours           int                          `json:"history_hours"`
	SampleIntervalMinutes  int                          `json:"sample_interval_minutes"`
	Forecast               StorageExpansionForecast     `json:"forecast"`
	Platform               SystemPlatformMetrics        `json:"platform"`
	CollectionTasks        []SystemCollectionTaskStatus `json:"collection_tasks"`
	RuntimeIntervalSeconds int                          `json:"runtime_interval_seconds"`
}
